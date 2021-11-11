package k8stree

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

func GetK8sTree(objects []interface{}) ([]*ObjectsTree, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(objects)
	if err != nil {
		return nil, err
	}

	unstructuredObjects := make([]*unstructured.Unstructured, 0, len(objects))
	err = json.Unmarshal(buf.Bytes(), &unstructuredObjects)
	if err != nil {
		return nil, err
	}

	sourceParents, unstructuredObjects := getSourceParents(unstructuredObjects)
	var objectsTrees []*ObjectsTree
	var sourceParentTree *ObjectsTree
	for _, sourceParent := range sourceParents {
		sourceParentTree, unstructuredObjects = createTrees(sourceParent, unstructuredObjects)
		objectsTrees = append(objectsTrees, sourceParentTree)
	}
	return objectsTrees, nil
}

func getSourceParents(objects []*unstructured.Unstructured) (
	[]*ObjectsTree, []*unstructured.Unstructured) {
	sourceParents := make([]*ObjectsTree, 0)

	for i, obj := range objects {
		objOwners := obj.GetOwnerReferences()

		if len(objOwners) == 0 {
			sourceParents = append(sourceParents, &ObjectsTree{
				uid: obj.GetUID(),
			})
			objects = append(objects[:i], objects[i+1:]...)
		}
	}
	return sourceParents, objects
}

type ObjectsTree struct {
	children []*ObjectsTree
	uid      types.UID
}

// newObjectDirectory builds object lookup and hierarchy.
func createTrees(objectsTree *ObjectsTree, objects []*unstructured.Unstructured) (
	*ObjectsTree, []*unstructured.Unstructured) {
	foundChildren := make([]*unstructured.Unstructured, 0)
	missingFatherChildren := objects[:0]

	for _, obj := range objects {
		if !containsChild(foundChildren, obj) {
			continue
		}
		ownerReference := obj.GetOwnerReferences()
		for _, ownerRef := range ownerReference {
			if ownerRef.UID == objectsTree.uid {
				childObj := &ObjectsTree{
					uid: obj.GetUID(),
				}
				objectsTree.children = append(objectsTree.children, childObj)
				_, objChildren := createTrees(childObj, missingFatherChildren)
				foundChildren = append(foundChildren, objChildren...)
				break
			}
		}
	}

	return objectsTree, foundChildren
}

func containsChild(children []*unstructured.Unstructured, child *unstructured.Unstructured) bool {
	for _, someChild := range children {
		if someChild.GetUID() == child.GetUID() {
			return true
		}
	}
	return false
}
