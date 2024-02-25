package v1alpha1

import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"aerf.io/provider-k8s/apis/common/v1alpha1"
	"aerf.io/provider-k8s/internal/controllers/generic"
)

// ObjectParameters are the configurable fields of a Object.
type ObjectParameters struct {
	// Raw YAML representation of the kubernetes object to be created.
	Manifest v1alpha1.Preserved `json:"manifest"`
}

// A ObjectSpec defines the desired state of a Object.
type ObjectSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ObjectParameters `json:"forProvider"`
	Readiness         Readiness        `json:"readiness,omitempty"`
}

type StatusWithObservedGeneration struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// A ObjectStatus represents the observed state of a Object.
type ObjectStatus struct {
	StatusWithObservedGeneration `json:",inline"`
	xpv1.ResourceStatus          `json:",inline"`
	AtProvider                   ObjectObservation `json:"atProvider,omitempty"`
}

// ObjectObservation are the observable fields of a Object.
type ObjectObservation struct {
	// Raw YAML representation of the remote object.
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	Manifest runtime.RawExtension `json:"manifest,omitempty"`
}

// ReadinessPolicy defines how the Object's readiness condition should be computed.
type ReadinessPolicy string

const (
	// ReadinessPolicySuccessfulCreate means the object is marked as ready when the
	// underlying external resource is successfully created.
	ReadinessPolicySuccessfulCreate ReadinessPolicy = "SuccessfulCreate"
	// ReadinessPolicyDeriveFromObject means the object is marked as ready if and only if the underlying
	// external resource is considered ready. Readiness is possible to compute if the object has `status` field with `conditions` subfield
	// and each condition has to have `type`, `status` and `meesage` fields.
	// Additionally, if the object has the optional `status.observedGeneration it is also used to compute its readiness
	ReadinessPolicyDeriveFromObject ReadinessPolicy = "DeriveFromObject"
	// ReadinessPolicyUseCELExpression means the object is marked ready if the celExpression returns true.
	ReadinessPolicyUseCELExpression ReadinessPolicy = "UseCELExpression"
)

// Readiness defines how the object's readiness condition should be computed,
// if not specified it will be considered ready as soon as the underlying external
// resource is considered up-to-date.
// +kubebuilder:validation:XValidation:rule="self.policy == 'UseCELExpression' ? has(self.celExpression) : !has(self.celExpression)",message="celExpression should be set only if policy is equal to UseCELExpression"
type Readiness struct {
	// `policy` defines how the Object's readiness condition should be computed.
	// +optional
	// +kubebuilder:validation:Enum=SuccessfulCreate;DeriveFromObject;UseCELExpression
	// +kubebuilder:default=SuccessfulCreate
	Policy ReadinessPolicy `json:"policy,omitempty"`
	// `celExpression` defines the CEL expression that should be executed to compute whether the Object is ready. It must return boolean value. See docs for examples.
	CELExpression string `json:"celExpression,omitempty"`
}

// +kubebuilder:object:root=true

// A Object is an provider Kubernetes API type
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="KIND",type="string",JSONPath=".spec.forProvider.manifest.kind"
// +kubebuilder:printcolumn:name="APIVERSION",type="string",JSONPath=".spec.forProvider.manifest.apiVersion",priority=1
// +kubebuilder:printcolumn:name="METANAME",type="string",JSONPath=".spec.forProvider.manifest.metadata.name",priority=1
// +kubebuilder:printcolumn:name="METANAMESPACE",type="string",JSONPath=".spec.forProvider.manifest.metadata.namespace",priority=1
// +kubebuilder:printcolumn:name="PROVIDERCONFIG",type="string",JSONPath=".spec.providerConfigRef.name"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,kubernetes}
type Object struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ObjectSpec   `json:"spec"`
	Status ObjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ObjectList contains a list of Object
type ObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Object `json:"items"`
}

// Object type metadata.
var (
	ObjectKind             = reflect.TypeOf(Object{}).Name()
	ObjectGroupKind        = schema.GroupKind{Group: Group, Kind: ObjectKind}.String()
	ObjectKindAPIVersion   = ObjectKind + "." + SchemeGroupVersion.String()
	ObjectGroupVersionKind = SchemeGroupVersion.WithKind(ObjectKind)
)

func init() {
	SchemeBuilder.Register(&Object{}, &ObjectList{})
}

var _ generic.ObservedGenerationSetter = &Object{}

func (o *Object) SetObservedGeneration(arg int64) {
	o.Status.ObservedGeneration = arg
}

func (o *Object) GetDesired() (*unstructured.Unstructured, error) {
	desired := &unstructured.Unstructured{}
	// if err := json.Unmarshal(o.Spec.ForProvider.Manifest.Rest, desired); err != nil {
	// 	return nil, errors.Wrap(err, "cannot unmarshal raw manifest")
	// }

	return desired, nil
}
