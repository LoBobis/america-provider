package datapower

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	xpevent "github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1alpha1 "github.com/crossplane/provider-america/apis/middleware/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-america/apis/v1alpha1"
	americaClient "github.com/crossplane/provider-america/internal/clients/america"
	"github.com/crossplane/provider-america/internal/controller/common"
)

const (
	errNotDataPower = "managed resource is not a DataPower custom resource"
)

// SetupGated adds a controller that reconciles DataPower managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options, webhookEventCh chan event.GenericEvent) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o, webhookEventCh); err != nil {
			panic(errors.Wrap(err, "cannot setup DataPower controller"))
		}
	}, v1alpha1.DataPowerGroupVersionKind)
	return nil
}

// Setup adds a controller that reconciles DataPower managed resources.
func Setup(mgr ctrl.Manager, o controller.Options, webhookEventCh chan event.GenericEvent) error {
	name := managed.ControllerName(v1alpha1.DataPowerGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			logger:             o.Logger,
			kube:               mgr.GetClient(),
			usage:              resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newAmericaClientFn: americaClient.NewClient}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(xpevent.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}

	if o.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(o.MetricOptions.MRMetrics))
	}

	if o.MetricOptions != nil && o.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.DataPowerList{}, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.DataPowerList")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.DataPowerGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.DataPower{}).
		WatchesRawSource(source.Channel(webhookEventCh, &handler.EnqueueRequestForObject{})).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	logger             logging.Logger
	kube               client.Client
	usage              *resource.ProviderConfigUsageTracker
	newAmericaClientFn func(log logging.Logger, creds string) (americaClient.Client, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.DataPower)
	if !ok {
		return nil, errors.New(errNotDataPower)
	}

	l, americaConfig, kube, america_client, err := common.ConnectExternal(
		c.logger, c.kube, c.usage, c.newAmericaClientFn, ctx, mg, cr,
	)
	if err != nil {
		return nil, err
	}

	return &external{
		logger:         l,
		americaConfig:  americaConfig,
		kube:           kube,
		america_client: america_client,
	}, nil
}

type external struct {
	logger         logging.Logger
	americaConfig  apisv1alpha1.AmericaConfig
	kube           client.Client
	america_client americaClient.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.DataPower)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDataPower)
	}

	return common.ObserveExternal(ctx, c.america_client, c.americaConfig, c.kube, cr, errNotDataPower)
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.DataPower)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDataPower)
	}
	return common.CreateExternal(ctx, c.america_client, c.americaConfig, c.kube, cr)
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.DataPower)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotDataPower)
	}

	return common.UpdateExternal(ctx, c.america_client, c.americaConfig, c.kube, cr)
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.DataPower)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotDataPower)
	}

	return common.DeleteExternal(c.america_client, c.americaConfig, cr)
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
