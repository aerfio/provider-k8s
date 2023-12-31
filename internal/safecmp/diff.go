package safecmp

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// cmp.Diff is NOT safe for production code (it may panic), as documented in
// https://pkg.go.dev/github.com/google/go-cmp/cmp.
// This version recovers the panic and returns it's error as the diff.
func Diff(x, y any, opts ...cmp.Option) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprintf("%v", r)
		}
	}()
	return cmp.Diff(x, y, opts...)
}

func DiffUnstructured(x, y *unstructured.Unstructured) string {
	xCopy := x.DeepCopy()
	yCopy := y.DeepCopy()

	cleanUnstructured(xCopy)
	cleanUnstructured(yCopy)

	return Diff(xCopy, yCopy)
}

func cleanUnstructured(obj *unstructured.Unstructured) {
	unstructured.RemoveNestedField(obj.Object, "status")
	obj.SetManagedFields(nil)
}
