package manageddiff

import (
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func safeCmp(x, y any) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = ""
		}
	}()
	return cmp.Diff(x, y)
}

func SafeDiff(x, y *unstructured.Unstructured) string {
	xCopy := x.DeepCopy()
	yCopy := y.DeepCopy()

	unstructured.RemoveNestedField(xCopy.Object, "status")
	xCopy.SetManagedFields(nil)

	unstructured.RemoveNestedField(yCopy.Object, "status")
	yCopy.SetManagedFields(nil)

	return safeCmp(xCopy, yCopy)
}
