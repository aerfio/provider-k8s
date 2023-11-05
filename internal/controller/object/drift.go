package object

import (
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// content of this file has been heavily influenced (copied even) from github.com/fluxcd/pkg/ssa library

// hasDrifted detects changes to metadata labels, annotations and spec.
func (e *external) hasDrifted(existingObject, dryRunObject *unstructured.Unstructured) bool {
	if dryRunObject.GetResourceVersion() == "" {
		return true
	}

	if !apiequality.Semantic.DeepEqual(dryRunObject.GetLabels(), existingObject.GetLabels()) {
		return true
	}

	if !apiequality.Semantic.DeepEqual(dryRunObject.GetAnnotations(), existingObject.GetAnnotations()) {
		return true
	}

	return hasObjectDrifted(dryRunObject, existingObject)
}

// hasObjectDrifted performs a semantic equality check of the given objects' spec
func hasObjectDrifted(existingObject, dryRunObject *unstructured.Unstructured) bool {
	existingObj := prepareObjectForDiff(existingObject)
	dryRunObj := prepareObjectForDiff(dryRunObject)

	return !apiequality.Semantic.DeepEqual(dryRunObj.Object, existingObj.Object)
}

// prepareObjectForDiff removes the metadata and status fields from the given object
func prepareObjectForDiff(object *unstructured.Unstructured) *unstructured.Unstructured {
	deepCopy := object.DeepCopy()
	unstructured.RemoveNestedField(deepCopy.Object, "metadata")
	unstructured.RemoveNestedField(deepCopy.Object, "status")
	return deepCopy
}
