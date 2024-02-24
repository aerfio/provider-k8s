package k8scmp

import (
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func DiffUnstructured(x, y *unstructured.Unstructured) string {
	xCopy := x.DeepCopy()
	yCopy := y.DeepCopy()

	cleanUnstructured(xCopy)
	cleanUnstructured(yCopy)

	return cmp.Diff(xCopy, yCopy)
}

func cleanUnstructured(obj *unstructured.Unstructured) {
	unstructured.RemoveNestedField(obj.Object, "status")
	obj.SetManagedFields(nil)
}
