package v1alpha1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPreserved_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *Preserved
		wantErr bool
	}{
		{
			name: "correctly unmarshals object that has both metadata, apiVersion and kind AND a field on root",
			data: []byte(`{
    "apiVersion": "v1",
    "data": {
        "some": "data"
    },
    "kind": "ConfigMap",
    "metadata": {
        "name": "output-report",
        "namespace": "default"
    }
}
`),
			want: &Preserved{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				PartialObjectMeta: PartialObjectMeta{
					Name:      "output-report",
					Namespace: "default",
				},
				Rest: map[string]any{
					"data": map[string]any{
						"some": "data",
					},
				},
			},
		},
		{
			name: "unmarshaling into rest field works with more complicated data",
			data: []byte(`{
    "apiVersion": "rbac.authorization.k8s.io/v1",
    "kind": "ClusterRole",
    "metadata": {
        "annotations": {
            "rbac.authorization.kubernetes.io/autoupdate": "true"
        },
        "creationTimestamp": "2023-09-18T14:20:08Z",
        "labels": {
            "kubernetes.io/bootstrapping": "rbac-defaults"
        },
        "name": "cluster-admin",
        "resourceVersion": "72",
        "uid": "b6c85305-bca2-41e7-98ad-35b3976c3d15"
    },
    "rules": [
        {
            "apiGroups": [
                "*"
            ],
            "resources": [
                "*"
            ],
            "verbs": [
                "*"
            ]
        },
        {
            "nonResourceURLs": [
                "*"
            ],
            "verbs": [
                "*"
            ]
        }
    ]
}
`),
			want: &Preserved{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				PartialObjectMeta: PartialObjectMeta{
					Name:        "cluster-admin",
					Labels:      map[string]string{"kubernetes.io/bootstrapping": "rbac-defaults"},
					Annotations: map[string]string{"rbac.authorization.kubernetes.io/autoupdate": "true"},
				},
				Rest: map[string]any{
					"rules": []any{
						map[string]any{"apiGroups": []any{"*"}, "resources": []any{"*"}, "verbs": []any{"*"}},
						map[string]any{"nonResourceURLs": []any{"*"}, "verbs": []any{"*"}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Preserved{}
			if err := p.UnmarshalJSON(tt.data); (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(p, tt.want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
