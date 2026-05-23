package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

type DataPowerProperties struct {
	Name string `json:"name"`
	Port int `json:"port"`
	Memory uint `json:"memory"`
	EnableTLS bool `json:"enable_t_l_s"`
}

type DataPowerCalculation struct {
	Status string `json:"status"`
}

// DataPowerParameters are the configurable fields of a DataPower.
type DataPowerParameters struct {
	// +kubebuilder:default="DataPowerAdvanced1_0"
	ResourceType            string `json:"resource_type"`
	DataPowerProperties     `json:"properties"`
	*DataPowerCalculation   `json:"resource_info"`
}

// A DataPowerSpec defines the desired state of a DataPower.
type DataPowerSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	Region                   string              `json:"region"`
	Environment              string              `json:"environment"`
	BundleID                 string              `json:"bundle_id"`
	ForProvider              DataPowerParameters  `json:"forProvider"`
}

// +kubebuilder:object:root=true

// A DataPower is an America managed resource.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,america}
type DataPower struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataPowerSpec  `json:"spec"`
	Status ResourceStatus `json:"status,omitempty"`
}

func (d *DataPower) GetName() string             { return d.Name }
func (d *DataPower) GetBundleID() string         { return d.Spec.BundleID }
func (d *DataPower) GetResourceType() string     { return d.Spec.ForProvider.ResourceType }
func (d *DataPower) GetEnvironment() string      { return d.Spec.Environment }
func (d *DataPower) GetRegion() string           { return d.Spec.Region }
func (d *DataPower) GetForProvider() interface{} { return &d.Spec.ForProvider }
func (d *DataPower) GetDeploymentID() string     { return d.Status.AtProvider.DeploymentID }

func (d *DataPower) SetDeploymentID(id string) { d.Status.AtProvider.DeploymentID = id }
func (d *DataPower) SetStatus(status string)   { d.Status.AtProvider.Status = status }
func (d *DataPower) SetAdditionalInfo(raw *runtime.RawExtension) {
	d.Status.AtProvider.AdditionalInfo = raw
}
func (d *DataPower) SetCondition(condition xpv1.Condition) { d.Status.SetConditions(condition) }

// +kubebuilder:object:root=true

// DataPowerList contains a list of DataPower
type DataPowerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataPower `json:"items"`
}

// DataPower type metadata.
var (
	DataPowerKind             = reflect.TypeOf(DataPower{}).Name()
	DataPowerGroupKind        = schema.GroupKind{Group: Group, Kind: DataPowerKind}.String()
	DataPowerKindAPIVersion   = DataPowerKind + "." + SchemeGroupVersion.String()
	DataPowerGroupVersionKind = SchemeGroupVersion.WithKind(DataPowerKind)
)

func init() {
	SchemeBuilder.Register(&DataPower{}, &DataPowerList{})
}
