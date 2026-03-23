package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// TopicParameters are the configurable fields of a Topic.
type TopicParameters struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	MaxMessageSize    uint   `json:"max_message_size"`
	MaxThroughputIn   uint   `json:"max_throughput_in"`
	MaxConsumerGroups uint   `json:"max_consumer_groups"`
	RetentionTime     int    `json:"retention_time"`
	NumOfPartitions   uint   `json:"num_of_partitions"`
}

// A TopicSpec defines the desired state of a Topic.
type TopicSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	Region                   string          `json:"region"`
	Environment              string          `json:"environment"`
	BundleID                 string          `json:"bundle_id"`
	ForProvider              TopicParameters `json:"forProvider"`
}

// +kubebuilder:object:root=true

// A Topic is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,america}
type Topic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TopicSpec     `json:"spec"`
	Status ResourceStatus `json:"status,omitempty"`

	// +kubebuilder:default="KafkaAdvancedConfig1_3"
	ResourceType string `json:"resource_type,omitempty"`
}

func (t *Topic) GetName() string             { return t.Name }
func (t *Topic) GetBundleID() string         { return t.Spec.BundleID }
func (t *Topic) GetResourceType() string     { return t.ResourceType }
func (t *Topic) GetEnvironment() string      { return t.Spec.Environment }
func (t *Topic) GetRegion() string           { return t.Spec.Region }
func (t *Topic) GetForProvider() interface{} { return &t.Spec.ForProvider }
func (t *Topic) GetDeploymentID() string     { return t.Status.AtProvider.DeploymentID }

func (t *Topic) SetDeploymentID(id string)                      { t.Status.AtProvider.DeploymentID = id }
func (t *Topic) SetStatus(status string)                        { t.Status.AtProvider.Status = status }
func (t *Topic) SetAdditionalInfo(raw *runtime.RawExtension)    { t.Status.AtProvider.AdditionalInfo = raw }
func (t *Topic) SetCondition(condition xpv1.Condition)          { t.Status.SetConditions(condition) }

// +kubebuilder:object:root=true

// TopicList contains a list of Topic
type TopicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Topic `json:"items"`
}

// Topic type metadata.
var (
	TopicKind             = reflect.TypeOf(Topic{}).Name()
	TopicGroupKind        = schema.GroupKind{Group: Group, Kind: TopicKind}.String()
	TopicKindAPIVersion   = TopicKind + "." + SchemeGroupVersion.String()
	TopicGroupVersionKind = SchemeGroupVersion.WithKind(TopicKind)
)

func init() {
	SchemeBuilder.Register(&Topic{}, &TopicList{})
}
