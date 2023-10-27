package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// LambdaForProvider are the configurable fields of a Lambda.
type LambdaForProvider struct {
	Handler string `json:"handler"`
}

// A LambdaSpec defines the desired state of a MyType.
type LambdaSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       LambdaForProvider `json:"forProvider"`
}

// A LambdaStatus represents the observed state of a MyType.
type LambdaStatus struct {
	ObservedGeneration  int64 `json:"observedGeneration,omitempty"`
	xpv1.ResourceStatus `json:",inline"`
}

// +kubebuilder:object:root=true

// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed}
type Lambda struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LambdaSpec   `json:"spec"`
	Status LambdaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LambdaList contains a list of Lambda
type LambdaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Lambda `json:"items"`
}

// MyType type metadata.
var (
	LambdaKind             = reflect.TypeOf(Lambda{}).Name()
	LambdaGroupKind        = schema.GroupKind{Group: Group, Kind: LambdaKind}.String()
	LambdaKindAPIVersion   = LambdaKind + "." + SchemeGroupVersion.String()
	LambdaGroupVersionKind = SchemeGroupVersion.WithKind(LambdaKind)
)

func init() {
	SchemeBuilder.Register(&Lambda{}, &LambdaList{})
}
