package k8stree

import (
	"github.com/infralight/k8s-collector/collector/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ObjectsTree struct {
	Children []ObjectsTree          `json:"children"`
	UID      string                 `json:"uid"`
	Kind     string                 `json:"kind"`
	Object   map[string]interface{} `json:"object"`
}

func GetK8sTree(objects []interface{}) ([]ObjectsTree, error) {
	unstructuredObjects := make([]unstructured.Unstructured, len(objects))
	for i, obj := range objects {
		unstructuredObjects[i] = unstructured.Unstructured{
			Object: obj.(k8s.KubernetesObject).Object.(map[string]interface{}),
		}
	}

	sourceParents, remainingUnstructuredObjects := getSourceParents(unstructuredObjects)
	var objectsTrees []ObjectsTree
	var sourceParentTree ObjectsTree
	for _, sourceParent := range sourceParents {
		sourceParentTree, remainingUnstructuredObjects, _ = createTrees(sourceParent, remainingUnstructuredObjects)
		objectsTrees = append(objectsTrees, sourceParentTree)
	}
	return objectsTrees, nil
}

func getSourceParents(objects []unstructured.Unstructured) (
	[]ObjectsTree, []unstructured.Unstructured) {
	sourceParents := make([]ObjectsTree, 0)
	remainingChildren := make([]unstructured.Unstructured, 0)

	for _, obj := range objects {
		objOwners := obj.GetOwnerReferences()

		if len(objOwners) == 0 {
			sourceParents = append(sourceParents, ObjectsTree{
				UID:    string(obj.GetUID()),
				Kind:   obj.GetKind(),
				Object: obj.Object,
			})
			continue
		}
		remainingChildren = append(remainingChildren, obj)
	}
	return sourceParents, remainingChildren
}

// newObjectDirectory builds object lookup and hierarchy.
func createTrees(objectsTree ObjectsTree, objects []unstructured.Unstructured) (
	ObjectsTree, []unstructured.Unstructured, []unstructured.Unstructured) {
	foundChildren := make([]unstructured.Unstructured, 0)
	remainingChildren := make([]unstructured.Unstructured, 0)
	objChildren := make([]unstructured.Unstructured, 0)
	var childTree ObjectsTree

	for _, obj := range objects {
		isChild := false
		if containsChild(foundChildren, obj) {
			continue
		}
		ownerReference := obj.GetOwnerReferences()
		for i, ownerRef := range ownerReference {
			if string(ownerRef.UID) == objectsTree.UID {
				if i == len(ownerReference)-1 {
					ownerReference = ownerReference[:i]
				} else {
					ownerReference = append(ownerReference[:i], ownerReference[i+1:]...)
				}
				obj.SetOwnerReferences(ownerReference)

				childObj := ObjectsTree{
					UID:    string(obj.GetUID()),
					Kind:   obj.GetKind(),
					Object: obj.Object,
				}
				childTree, remainingChildren, objChildren = createTrees(childObj, remainingChildren)
				objectsTree.Children = append(objectsTree.Children, childTree)
				foundChildren = append(foundChildren, objChildren...)
				isChild = true
				break
			}
		}

		if !isChild || len(ownerReference) != 0 {
			remainingChildren = append(remainingChildren, obj)
		}
	}

	return objectsTree, remainingChildren, foundChildren
}

func containsChild(children []unstructured.Unstructured, child unstructured.Unstructured) bool {
	for _, someChild := range children {
		if someChild.GetUID() == child.GetUID() {
			return true
		}
	}
	return false
}
