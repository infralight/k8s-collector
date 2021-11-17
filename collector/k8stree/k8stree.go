package k8stree

import (
	"github.com/infralight/k8s-collector/collector/k8s"
	"github.com/thoas/go-funk"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	funk.ForEach(objects, func(obj interface{}) {
		unstructuredObjects = append(unstructuredObjects, unstructured.Unstructured{
			Object: obj.(k8s.KubernetesObject).Object.(map[string]interface{}),
		})
	})

	sourceParents, remainingUnstructuredObjects := getSourceParents(unstructuredObjects)
	var objectsTrees []ObjectsTree
	var sourceParentTree ObjectsTree
	var foundChildren []unstructured.Unstructured
	for _, sourceParent := range sourceParents {
		sourceParentTree, foundChildren = createTrees(sourceParent, remainingUnstructuredObjects)
		remainingUnstructuredObjects = subtractUnstructuredObjects(remainingUnstructuredObjects, foundChildren)
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

func createTrees(objectsTree ObjectsTree, objects []unstructured.Unstructured) (
	ObjectsTree, []unstructured.Unstructured) {
	foundChildren := make([]unstructured.Unstructured, 0)
	objChildren := make([]unstructured.Unstructured, 0)
	var childTree ObjectsTree

	for _, obj := range objects {
		ownerReference := obj.GetOwnerReferences()
		for _, ownerRef := range ownerReference {
			if string(ownerRef.UID) == objectsTree.UID {
				ownerReference = subtractOwnerReferences(ownerReference, []v1.OwnerReference{ownerRef})
				obj.SetOwnerReferences(ownerReference)

				if len(ownerReference) == 0 {
					foundChildren = append(foundChildren, obj)
				}
				remainingChildren := subtractUnstructuredObjects(objects, foundChildren)

				childObj := ObjectsTree{
					UID:    string(obj.GetUID()),
					Kind:   obj.GetKind(),
					Object: obj.Object,
				}
				childTree, objChildren = createTrees(childObj, remainingChildren)
				objectsTree.Children = append(objectsTree.Children, childTree)
				foundChildren = append(foundChildren, objChildren...)
				break
			}
		}
	}

	return objectsTree, foundChildren
}

func subtractUnstructuredObjects(a, b []unstructured.Unstructured) []unstructured.Unstructured {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[string(x.GetUID())] = struct{}{}
	}
	var diff []unstructured.Unstructured
	for _, x := range a {
		if _, found := mb[string(x.GetUID())]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func subtractOwnerReferences(a, b []v1.OwnerReference) []v1.OwnerReference {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[string(x.UID)] = struct{}{}
	}
	var diff []v1.OwnerReference
	for _, x := range a {
		if _, found := mb[string(x.UID)]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
