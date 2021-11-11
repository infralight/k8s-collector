package k8stree

import (
	"github.com/infralight/k8s-collector/collector/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

type ObjectsTree struct {
	children []*ObjectsTree
	uid      types.UID
	kind     string
	object   unstructured.Unstructured
}

func GetK8sTree(objects []interface{}) ([]*ObjectsTree, error) {
	unstructuredObjects := make([]*unstructured.Unstructured, len(objects))
	for i, obj := range objects {
		unstructuredObjects[i] = &unstructured.Unstructured{
			Object: obj.(k8s.KubernetesObject).Object.(map[string]interface{}),
		}
	}

	sourceParents, remainingUnstructuredObjects := getSourceParents(unstructuredObjects)
	var objectsTrees []*ObjectsTree
	var sourceParentTree *ObjectsTree
	for _, sourceParent := range sourceParents {
		sourceParentTree, remainingUnstructuredObjects, _ = createTrees(sourceParent, remainingUnstructuredObjects)
		objectsTrees = append(objectsTrees, sourceParentTree)
	}
	return objectsTrees, nil
}

func getSourceParents(objects []*unstructured.Unstructured) (
	[]*ObjectsTree, []*unstructured.Unstructured) {
	sourceParents := make([]*ObjectsTree, 0)
	remainingChildren := make([]*unstructured.Unstructured, 0)

	for _, obj := range objects {
		objOwners := obj.GetOwnerReferences()

		if len(objOwners) == 0 {
			sourceParents = append(sourceParents, &ObjectsTree{
				uid:    obj.GetUID(),
				kind:   obj.GetKind(),
				object: *obj,
			})
			continue
		}
		remainingChildren = append(remainingChildren, obj)
	}
	return sourceParents, remainingChildren
}

// newObjectDirectory builds object lookup and hierarchy.
func createTrees(objectsTree *ObjectsTree, objects []*unstructured.Unstructured) (
	*ObjectsTree, []*unstructured.Unstructured, []*unstructured.Unstructured) {
	foundChildren := make([]*unstructured.Unstructured, 0)
	remainingChildren := make([]*unstructured.Unstructured, 0)
	objChildren := make([]*unstructured.Unstructured, 0)

	for _, obj := range objects {
		isChild := false
		if containsChild(foundChildren, obj) {
			continue
		}
		ownerReference := obj.GetOwnerReferences()
		for _, ownerRef := range ownerReference {
			if ownerRef.UID == objectsTree.uid {
				childObj := &ObjectsTree{
					uid:    obj.GetUID(),
					kind:   obj.GetKind(),
					object: *obj,
				}
				objectsTree.children = append(objectsTree.children, childObj)
				_, remainingChildren, objChildren = createTrees(childObj, remainingChildren)
				foundChildren = append(foundChildren, objChildren...)
				isChild = true
				break
			}
		}

		if !isChild || len(ownerReference) > 1 {
			remainingChildren = append(remainingChildren, obj)
		}
	}

	return objectsTree, remainingChildren, foundChildren
}

func containsChild(children []*unstructured.Unstructured, child *unstructured.Unstructured) bool {
	for _, someChild := range children {
		if someChild.GetUID() == child.GetUID() {
			return true
		}
	}
	return false
}
