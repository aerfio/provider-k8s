package manageddiff

import (
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func safeCmp(x any, y any) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = ""
		}
	}()
	return cmp.Diff(x, y)
}

func SafeDiff(x client.Object, y client.Object) string {
	xunstr := &unstructured.Unstructured{}
	yunstr := &unstructured.Unstructured{}
	var err error
	xunstr.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(x)
	if err != nil {
		return ""
	}

	yunstr.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(y)
	if err != nil {
		return ""
	}
	unstructured.RemoveNestedField(xunstr.Object, "status")
	xunstr.SetManagedFields(nil)

	unstructured.RemoveNestedField(yunstr.Object, "status")
	yunstr.SetManagedFields(nil)

	return safeCmp(xunstr, yunstr)
}
