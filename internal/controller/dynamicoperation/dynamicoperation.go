package dynamicoperation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	v1alpha1 "github.com/crossplane/provider-america/apis/middleware/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-america/apis/v1alpha1"
	americaClient "github.com/crossplane/provider-america/internal/clients/america"
)

const (
	requeuePending = 15 * time.Second
)

// Setup adds a controller that reconciles DynamicOperation resources.
// Unlike standard managed resources, this controller uses a predicate filter
// to stop reconciling once the operation reaches a terminal state (Completed/Failed).
func Setup(mgr ctrl.Manager, logger logging.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("DynamicOperation").
		For(&v1alpha1.DynamicOperation{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			op, ok := obj.(*v1alpha1.DynamicOperation)
			if !ok {
				return true
			}
			// Only reconcile if the operation is NOT in a terminal state.
			status := op.Status.AtProvider.OperationStatus
			return status != americaClient.OperationStatuses.COMPLETED &&
				status != americaClient.OperationStatuses.FAILED
		})).
		Complete(&reconciler{
			kube:               mgr.GetClient(),
			logger:             logger,
			newAmericaClientFn: americaClient.NewClient,
		})
}

type reconciler struct {
	kube               client.Client
	logger             logging.Logger
	newAmericaClientFn func(log logging.Logger, creds string) (americaClient.Client, error)
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.logger.WithValues("dynamicoperation", req.NamespacedName)

	// Fetch the DynamicOperation CR.
	op := &v1alpha1.DynamicOperation{}
	if err := r.kube.Get(ctx, req.NamespacedName, op); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Connect to the America API.
	americaConfig, americaC, err := r.connect(ctx, op, log)
	if err != nil {
		log.Info("Cannot connect to America", "error", err)
		op.Status.SetConditions(xpv1.ReconcileError(err))
		_ = r.kube.Status().Update(ctx, op)
		return ctrl.Result{RequeueAfter: requeuePending}, nil
	}

	// If no operationID yet, create the operation.
	if op.Status.AtProvider.OperationID == "" {
		return r.create(ctx, op, americaConfig, americaC, log)
	}

	// Otherwise, poll for the operation status.
	return r.observe(ctx, op, americaConfig, americaC, log)
}

func (r *reconciler) create(ctx context.Context, op *v1alpha1.DynamicOperation, americaConfig apisv1alpha1.AmericaConfig, americaC americaClient.Client, log logging.Logger) (ctrl.Result, error) {
	log.Info("Creating DynamicOperation", "operationType", op.Spec.ForProvider.OperationType)

	// Serialize payload to string.
	payload := "{}"
	if op.Spec.ForProvider.Payload != nil && op.Spec.ForProvider.Payload.Raw != nil {
		payload = string(op.Spec.ForProvider.Payload.Raw)
	}

	resp, err := americaC.CreateDynamicOperation(
		ctx,
		americaConfig.AmericaURL,
		op.Spec.ForProvider.OperationType,
		payload,
	)
	if err != nil {
		log.Info("Failed to create dynamic operation", "error", err)
		op.Status.AtProvider.OperationStatus = americaClient.OperationStatuses.FAILED
		op.Status.SetConditions(xpv1.ReconcileError(errors.Wrap(err, "cannot create dynamic operation")))
		_ = r.kube.Status().Update(ctx, op)
		return ctrl.Result{}, nil // Don't requeue on failure — terminal state.
	}

	op.Status.AtProvider.OperationID = resp.OperationID
	op.Status.AtProvider.OperationStatus = americaClient.OperationStatuses.PENDING
	op.Status.SetConditions(xpv1.ReconcileSuccess())
	op.Status.SetConditions(xpv1.Creating())
	if err := r.kube.Status().Update(ctx, op); err != nil {
		return ctrl.Result{}, err
	}

	fmt.Printf("DynamicOperation created: operationID=%s\n", resp.OperationID)
	return ctrl.Result{RequeueAfter: requeuePending}, nil
}

func (r *reconciler) observe(ctx context.Context, op *v1alpha1.DynamicOperation, americaConfig apisv1alpha1.AmericaConfig, americaC americaClient.Client, log logging.Logger) (ctrl.Result, error) {
	log.Info("Polling DynamicOperation", "operationID", op.Status.AtProvider.OperationID)

	resp, err := americaC.GetDynamicOperation(ctx, americaConfig.AmericaURL, op.Status.AtProvider.OperationID)
	if err != nil {
		log.Info("Failed to get dynamic operation status", "error", err)
		op.Status.SetConditions(xpv1.ReconcileError(errors.Wrap(err, "cannot get dynamic operation")))
		_ = r.kube.Status().Update(ctx, op)
		return ctrl.Result{RequeueAfter: requeuePending}, nil
	}

	op.Status.AtProvider.OperationStatus = resp.Status

	// Store result if present.
	if resp.Result != nil {
		resultJSON, err := json.Marshal(resp.Result)
		if err == nil {
			op.Status.AtProvider.Result = &runtime.RawExtension{Raw: resultJSON}
		}
	}

	switch resp.Status {
	case americaClient.OperationStatuses.COMPLETED:
		log.Info("DynamicOperation completed", "operationID", op.Status.AtProvider.OperationID)
		op.Status.SetConditions(xpv1.Available())
		op.Status.SetConditions(xpv1.ReconcileSuccess())
		if err := r.kube.Status().Update(ctx, op); err != nil {
			return ctrl.Result{}, err
		}
		// Terminal state — don't requeue. The predicate will also filter future events.
		return ctrl.Result{}, nil

	case americaClient.OperationStatuses.FAILED:
		log.Info("DynamicOperation failed", "operationID", op.Status.AtProvider.OperationID)
		op.Status.SetConditions(xpv1.Unavailable())
		op.Status.SetConditions(xpv1.ReconcileError(errors.New("dynamic operation failed")))
		if err := r.kube.Status().Update(ctx, op); err != nil {
			return ctrl.Result{}, err
		}
		// Terminal state — don't requeue.
		return ctrl.Result{}, nil

	default:
		// Still pending — requeue to poll again.
		op.Status.SetConditions(xpv1.ReconcileSuccess())
		_ = r.kube.Status().Update(ctx, op)
		return ctrl.Result{RequeueAfter: requeuePending}, nil
	}
}

// connect reads the ProviderConfig and creates an America client.
func (r *reconciler) connect(ctx context.Context, op *v1alpha1.DynamicOperation, log logging.Logger) (apisv1alpha1.AmericaConfig, americaClient.Client, error) {
	ref := op.GetProviderConfigReference()
	if ref == nil {
		return apisv1alpha1.AmericaConfig{}, nil, errors.New("no providerConfigRef set")
	}

	var cd apisv1alpha1.ProviderCredentials
	var am apisv1alpha1.AmericaConfig

	switch ref.Kind {
	case "ProviderConfig":
		pc := &apisv1alpha1.ProviderConfig{}
		if err := r.kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: op.GetNamespace()}, pc); err != nil {
			return apisv1alpha1.AmericaConfig{}, nil, errors.Wrap(err, "cannot get ProviderConfig")
		}
		cd = pc.Spec.Credentials
		am = pc.Spec.AmericaConfig
	case "ClusterProviderConfig":
		cpc := &apisv1alpha1.ClusterProviderConfig{}
		if err := r.kube.Get(ctx, types.NamespacedName{Name: ref.Name}, cpc); err != nil {
			return apisv1alpha1.AmericaConfig{}, nil, errors.Wrap(err, "cannot get ClusterProviderConfig")
		}
		cd = cpc.Spec.Credentials
		am = cpc.Spec.AmericaConfig
	default:
		return apisv1alpha1.AmericaConfig{}, nil, errors.Errorf("unsupported provider config kind: %s", ref.Kind)
	}

	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, r.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return apisv1alpha1.AmericaConfig{}, nil, errors.Wrap(err, "cannot get credentials")
	}

	ac, err := r.newAmericaClientFn(log, string(data))
	if err != nil {
		return apisv1alpha1.AmericaConfig{}, nil, errors.Wrap(err, "cannot create America client")
	}

	return am, ac, nil
}
