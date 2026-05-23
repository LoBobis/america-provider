package america

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	xpevent "github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	apisv1alpha1 "github.com/crossplane/provider-america/apis/v1alpha1"
	americaClient "github.com/crossplane/provider-america/internal/clients/america"
	"github.com/crossplane/provider-america/internal/controller/common"
)

// ResourceConfig holds all type-specific values needed to register a controller
// for a Deployable managed resource.
type ResourceConfig struct {
	Kind             string
	GroupKind        string
	GroupVersionKind schema.GroupVersionKind
	NewObject        func() common.Deployable
	NewObjectList    func() resource.ManagedList
	CastManaged      func(resource.Managed) (common.Deployable, error)
}

// SetupGated registers a gated controller for the given resource config.
func SetupGated(mgr ctrl.Manager, o controller.Options, webhookEventCh chan event.GenericEvent, cfg ResourceConfig) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o, webhookEventCh, cfg); err != nil {
			panic(errors.Wrapf(err, "cannot setup %s controller", cfg.Kind))
		}
	}, cfg.GroupVersionKind)
	return nil
}

// Setup registers a managed reconciler controller for the given resource config.
func Setup(mgr ctrl.Manager, o controller.Options, webhookEventCh chan event.GenericEvent, cfg ResourceConfig) error {
	name := managed.ControllerName(cfg.GroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			logger:             o.Logger,
			kube:               mgr.GetClient(),
			usage:              resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newAmericaClientFn: americaClient.NewClient,
			cfg:                cfg,
		}),
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
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, cfg.NewObjectList(), o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrapf(err, "cannot register MR state metrics recorder for kind %s", cfg.Kind)
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(cfg.GroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(cfg.NewObject().(client.Object)).
		WatchesRawSource(source.Channel(webhookEventCh, &handler.EnqueueRequestForObject{})).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	logger             logging.Logger
	kube               client.Client
	usage              *resource.ProviderConfigUsageTracker
	newAmericaClientFn func(log logging.Logger, creds string) (americaClient.Client, error)
	cfg                ResourceConfig
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, err := c.cfg.CastManaged(mg)
	if err != nil {
		return nil, err
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
		cfg:            c.cfg,
	}, nil
}

type external struct {
	logger         logging.Logger
	americaConfig  apisv1alpha1.AmericaConfig
	kube           client.Client
	america_client americaClient.Client
	cfg            ResourceConfig
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, err := e.cfg.CastManaged(mg)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	errNotFound := fmt.Sprintf("managed resource is not a %s custom resource", e.cfg.Kind)
	return common.ObserveExternal(ctx, e.america_client, e.americaConfig, e.kube, cr, errNotFound)
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, err := e.cfg.CastManaged(mg)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	return common.CreateExternal(ctx, e.america_client, e.americaConfig, e.kube, cr)
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, err := e.cfg.CastManaged(mg)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	return common.UpdateExternal(ctx, e.america_client, e.americaConfig, e.kube, cr)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, err := e.cfg.CastManaged(mg)
	if err != nil {
		return managed.ExternalDelete{}, err
	}
	return common.DeleteExternal(e.america_client, e.americaConfig, cr)
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}
