package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Type=object
// +kubebuilder:pruning:PreserveUnknownFields
// +kubebuilder:validation:XValidation:rule="self.kind == oldSelf.kind",message="Kind is immutable"
// +kubebuilder:validation:XValidation:rule="!(has(self.metadata.generateName))",message="generateName is disallowed"
// +kubebuilder:validation:XValidation:rule="self.apiVersion == oldSelf.apiVersion",message="APIVersion is immutable"
// +kubebuilder:validation:XValidation:rule="self.metadata.name == oldSelf.metadata.name",message="metadata.name is immutable"
type Preserved struct {
	metav1.TypeMeta   `json:",inline"`
	PartialObjectMeta `json:"metadata"`
	Rest              map[string]any `json:"-"`
}

type PartialObjectMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type unmarshalObj struct {
	PartialObjectMeta `json:"metadata"`
}

func (p *Preserved) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &p.Rest); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &p.TypeMeta); err != nil {
		return err
	}
	unmarshalMetaWrapper := &unmarshalObj{}

	if err := json.Unmarshal(data, unmarshalMetaWrapper); err != nil {
		return err
	}
	p.PartialObjectMeta = unmarshalMetaWrapper.PartialObjectMeta

	delete(p.Rest, "kind")
	delete(p.Rest, "apiVersion")
	delete(p.Rest, "metadata")

	return nil
}

func (p *Preserved) MarshalJSON() ([]byte, error) {
	full := make(map[string]any, len(p.Rest))
	for k, v := range p.Rest {
		full[k] = v
	}
	full["kind"] = p.Kind
	full["apiVersion"] = p.APIVersion
	full["metadata"] = p.PartialObjectMeta
	return json.Marshal(full)
}

func (p *Preserved) DeepCopyInto(out *Preserved) {
	*out = *p
}

func (p *Preserved) DeepCopy() *Preserved {
	if p == nil {
		return nil
	}
	b, err := p.MarshalJSON()
	if err != nil {
		panic(err)
	}
	out := &Preserved{}
	if err := json.Unmarshal(b, out); err != nil {
		panic(err)
	}

	return out
}
