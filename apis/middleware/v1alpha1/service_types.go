package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"

	americaClient "github.com/crossplane/provider-template/internal/clients/america"
)

// ServiceParameters are the configurable fields of a Service.
type ServiceParameters struct {
	ServiceName     string `json:"serviceName"`
	ServicePassword string `json:"servicePassword"`
}

// ServiceObservation are the observable fields of a Service.

// A ServiceSpec defines the desired state of a Service.
type ServiceSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	BundleID                 string            `json:"bundle_id"`
	ForProvider              ServiceParameters `json:"forProvider"`
}

// +kubebuilder:object:root=true

// A Service is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,america}
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceSpec     `json:"spec"`
	Status ResourceStatus  `json:"status,omitempty"`

	// +kubebuilder:default="service-4.0.1"
	ResourceType string `json:"resource_type,omitempty"`
}

func (s *Service) GetName() string             { return s.Name }
func (s *Service) GetBundleID() string         { return s.Spec.BundleID }
func (s *Service) GetResourceType() string     { return s.ResourceType }
func (s *Service) GetEnvironment() string      { return americaClient.AmericaEnv.GLOBAL }
func (s *Service) GetRegion() string           { return americaClient.AmericaRegionsName.DEFAULT }
func (s *Service) GetForProvider() interface{} { return s.Spec.ForProvider }
func (s *Service) GetDeploymentID() string     { return s.Status.AtProvider.DeploymentID }

func (s *Service) SetDeploymentID(id string)                   { s.Status.AtProvider.DeploymentID = id }
func (s *Service) SetStatus(status string)                     { s.Status.AtProvider.Status = status }
func (s *Service) SetAdditionalInfo(raw *runtime.RawExtension) { s.Status.AtProvider.AdditionalInfo = raw }
func (s *Service) SetCondition(condition xpv1.Condition)       { s.Status.SetConditions(condition) }

// +kubebuilder:object:root=true

// ServiceList contains a list of Service
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

// Service type metadata.
var (
	ServiceKind             = reflect.TypeOf(Service{}).Name()
	ServiceGroupKind        = schema.GroupKind{Group: Group, Kind: ServiceKind}.String()
	ServiceKindAPIVersion   = ServiceKind + "." + SchemeGroupVersion.String()
	ServiceGroupVersionKind = SchemeGroupVersion.WithKind(ServiceKind)
)

func init() {
	SchemeBuilder.Register(&Service{}, &ServiceList{})
}
