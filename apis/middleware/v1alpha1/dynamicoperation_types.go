package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// DynamicOperationParameters are the configurable fields of a DynamicOperation.
type DynamicOperationParameters struct {
	// OperationType is the type of operation to perform in America.
	OperationType string `json:"operationType"`

	// Payload is the JSON payload to send with the operation.
	// +kubebuilder:pruning:PreserveUnknownFields
	Payload *runtime.RawExtension `json:"payload,omitempty"`
}

// DynamicOperationAtProvider represents the observed state of a DynamicOperation.
type DynamicOperationAtProvider struct {
	// OperationID is the unique identifier returned by the America API.
	OperationID string `json:"operationID,omitempty"`

	// OperationStatus is the current status of the operation (Pending, Completed, Failed).
	OperationStatus string `json:"operationStatus,omitempty"`

	// Result contains the response from the America API after the operation completes.
	// +kubebuilder:pruning:PreserveUnknownFields
	Result *runtime.RawExtension `json:"result,omitempty"`
}

// A DynamicOperationSpec defines the desired state of a DynamicOperation.
type DynamicOperationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	Environment              string                     `json:"environment"`
	Region                   string                     `json:"region"`
	BundleID                 string                     `json:"bundle_id"`
	ForProvider              DynamicOperationParameters  `json:"forProvider"`
}

// DynamicOperationStatus represents the status of a DynamicOperation.
type DynamicOperationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DynamicOperationAtProvider `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A DynamicOperation is a fire-and-forget operation in America.
// Once completed, the controller stops reconciling — similar to a Kubernetes Job.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="OPERATION-STATUS",type="string",JSONPath=".status.atProvider.operationStatus"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,america}
type DynamicOperation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DynamicOperationSpec   `json:"spec"`
	Status DynamicOperationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DynamicOperationList contains a list of DynamicOperation
type DynamicOperationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DynamicOperation `json:"items"`
}

// DynamicOperation type metadata.
var (
	DynamicOperationKind             = reflect.TypeOf(DynamicOperation{}).Name()
	DynamicOperationGroupKind        = schema.GroupKind{Group: Group, Kind: DynamicOperationKind}.String()
	DynamicOperationKindAPIVersion   = DynamicOperationKind + "." + SchemeGroupVersion.String()
	DynamicOperationGroupVersionKind = SchemeGroupVersion.WithKind(DynamicOperationKind)
)

func init() {
	SchemeBuilder.Register(&DynamicOperation{}, &DynamicOperationList{})
}
