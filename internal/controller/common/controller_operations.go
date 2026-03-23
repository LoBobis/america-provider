package common

import (
	"context"
	"encoding/json"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apisv1alpha1 "github.com/crossplane/provider-template/apis/v1alpha1"
	americaClient "github.com/crossplane/provider-template/internal/clients/america"
)

const (
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCPC       = "cannot get ClusterProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create new Service"
)

// Deployable is an interface for resources that can be deployed via America.
type Deployable interface {
	resource.Managed
	GetName() string
	GetBundleID() string
	GetResourceType() string
	GetEnvironment() string
	GetRegion() string
	GetForProvider() interface{}
	GetDeploymentID() string
	SetDeploymentID(string)
	SetStatus(string)
	SetAdditionalInfo(*runtime.RawExtension)
	SetCondition(xpv1.Condition)
}

// IsCreated checks if the resource has been created (has a deployment ID).
func IsCreated(cr Deployable) bool {
	return cr.GetDeploymentID() != ""
}

// updateCRStatus updates the CR status in Kubernetes.
func updateCRStatus(ctx context.Context, kube client.Client, cr Deployable) error {
	return kube.Status().Update(ctx, cr)
}

// CreateExternal creates a new deployment in America.
func CreateExternal(ctx context.Context, america_client americaClient.Client, americaConfig apisv1alpha1.AmericaConfig, kube client.Client,
	cr Deployable) (managed.ExternalCreation, error) {

	fmt.Println("Started Create Operation For Resource With Name:", cr.GetName())
	resp, err := america_client.CreateDeployment(
		cr.GetName(),
		americaConfig.JWTKey,
		americaConfig.AmericaURL,
		americaConfig.ProjectID,
		cr.GetBundleID(),
		cr.GetResourceType(),
		cr.GetEnvironment(),
		cr.GetRegion(),
		cr.GetForProvider(),
	)

	if err != nil {
		fmt.Println("Error Creating Resource:", err)
		return managed.ExternalCreation{}, fmt.Errorf("America create failed for %s: %w", cr.GetName(), err)
	}
	cr.SetDeploymentID(resp.LatestOperation.DeploymentUUID)
	cr.SetStatus(americaClient.DeploymentStatuses.PENDING)

	if err := updateCRStatus(ctx, kube, cr); err != nil {
		return managed.ExternalCreation{}, err
	}

	fmt.Printf("Creating: %+v \n", resp.LatestOperation.DeploymentUUID)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// DeleteExternal deletes a deployment in America.
func DeleteExternal(america_client americaClient.Client, americaConfig apisv1alpha1.AmericaConfig,
	cr Deployable) (managed.ExternalDelete, error) {

	fmt.Println("Started Delete Operation For Resource With Name:", cr.GetName())
	america_client.DeleteDeployment(americaConfig.JWTKey, americaConfig.AmericaURL, cr.GetDeploymentID())

	return managed.ExternalDelete{}, nil
}

// ObserveExternal observes the current state of a deployment in America.
func ObserveExternal(ctx context.Context, america_client americaClient.Client, americaConfig apisv1alpha1.AmericaConfig, kube client.Client,
	cr Deployable, errNotFound string) (managed.ExternalObservation, error) {
	fmt.Printf("Started Observe Operation For Resource of Type %T With Name: %s \n", cr, cr.GetName())
	if !IsCreated(cr) {
		fmt.Println("Resource is not created", cr.GetName())
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	deploymentID := cr.GetDeploymentID()
	fmt.Println("Deployment id:", deploymentID)
	url := americaConfig.AmericaURL
	resp, err := america_client.GetDeployment(deploymentID, americaConfig.JWTKey, url)

	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return managed.ExternalObservation{}, errors.New(errNotFound)
	}

	cr.SetCondition(xpv1.Creating())
	cr.SetStatus(resp.Status)

	if resp.Status == americaClient.DeploymentStatuses.CREATED {
		ad, err := json.Marshal(resp.AdditionalInfo)
		if err != nil {
		}

		raw := runtime.RawExtension{
			Raw: ad,
		}
		cr.SetAdditionalInfo(&raw)
		cr.SetCondition(xpv1.Available())
	}

	if err := updateCRStatus(ctx, kube, cr); err != nil {
		return managed.ExternalObservation{}, err
	}

	// if resp.Status == americaClient.DeploymentStatuses.ERROR {
	// 	return managed.ExternalObservation{
	// 		ResourceExists:    false,
	// 		ResourceUpToDate:  false,
	// 		ConnectionDetails: managed.ConnectionDetails{},
	// 	}, nil
	// }

	if resp.Status == americaClient.DeploymentStatuses.DELETED {
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  true,
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// ConnectExternal establishes a connection to the America API.
func ConnectExternal(logger logging.Logger, kube client.Client, usage *resource.ProviderConfigUsageTracker,
	newAmericaClientFn func(log logging.Logger, creds string) (americaClient.Client, error),
	ctx context.Context, mg resource.Managed, cr Deployable) (logging.Logger, apisv1alpha1.AmericaConfig, client.Client, americaClient.Client, error) {
	fmt.Printf("Started Connect Operation For Resource of Type %T With Name: %s \n", cr, cr.GetName())

	// Switch to ModernManaged resource to get ProviderConfigRef
	m := mg.(resource.ModernManaged)

	if err := usage.Track(ctx, m); err != nil {
		return nil, apisv1alpha1.AmericaConfig{}, nil, nil, errors.Wrap(err, errTrackPCUsage)
	}

	var cd apisv1alpha1.ProviderCredentials

	ref := m.GetProviderConfigReference()
	am := apisv1alpha1.AmericaConfig{}
	switch ref.Kind {
	case "ProviderConfig":
		pc := &apisv1alpha1.ProviderConfig{}
		if err := kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: m.GetNamespace()}, pc); err != nil {
			return nil, apisv1alpha1.AmericaConfig{}, nil, nil, errors.Wrap(err, errGetPC)
		}
		cd = pc.Spec.Credentials
		am = pc.Spec.AmericaConfig
	case "ClusterProviderConfig":
		cpc := &apisv1alpha1.ClusterProviderConfig{}
		if err := kube.Get(ctx, types.NamespacedName{Name: ref.Name}, cpc); err != nil {
			return nil, apisv1alpha1.AmericaConfig{}, nil, nil, errors.Wrap(err, errGetCPC)
		}
		cd = cpc.Spec.Credentials
		am = cpc.Spec.AmericaConfig
	default:
		return nil, apisv1alpha1.AmericaConfig{}, nil, nil, errors.Errorf("unsupported provider config kind: %s", ref.Kind)
	}

	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, apisv1alpha1.AmericaConfig{}, nil, nil, errors.Wrap(err, errGetCreds)
	}

	creds := string(data)
	l := logger.WithValues("XT", cr, cr.GetName())
	a_client, err := newAmericaClientFn(l, creds)
	if err != nil {
		return nil, apisv1alpha1.AmericaConfig{}, nil, nil, errors.Wrap(err, errNewClient)
	}

	return l, am, kube, a_client, nil
}
