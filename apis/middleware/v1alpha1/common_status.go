package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AtProviderStatus represents the observed state of a resource managed by America.
type AtProviderStatus struct {
	// DeploymentID is the unique identifier for the deployment in America.
	DeploymentID string `json:"deploymentID,omitempty"`

	// Status is the current status of the deployment.
	Status string `json:"status,omitempty"`

	// AdditionalInfo contains extra information returned by America.
	AdditionalInfo *runtime.RawExtension `json:"additionalInfo,omitempty"`
}

// ResourceStatus represents the status of a managed resource with AtProvider.
type ResourceStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider contains the observed state of the resource.
	AtProvider AtProviderStatus `json:"atProvider,omitempty"`
}
