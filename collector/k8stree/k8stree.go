package k8stree

import (
	"fmt"
	"github.com/infralight/k8s-collector/collector/k8s"
	"github.com/thoas/go-funk"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
)

type ObjectsTree struct {
	Children []ObjectsTree          `json:"children"`
	UID      string                 `json:"uid"`
	Kind     string                 `json:"kind"`
	Name     string                 `json:"name,omitempty"`
	Object   map[string]interface{} `json:"object"`
}

func GetK8sTree(objects []interface{}) ([]ObjectsTree, error) {
	var unstructuredObjects []unstructured.Unstructured
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
	specialParents := getSpecialParents(objects, []string{"Service", "StatefulSet", "PersistentVolumeClaim"})
	sourceParents := make([]ObjectsTree, 0)
	remainingChildren := make([]unstructured.Unstructured, 0)

	for _, obj := range objects {
		objOwners := obj.GetOwnerReferences()

		if len(objOwners) == 0 {
			isSpecialChildren, newObjOwners := specialChildren(obj, specialParents)
			if isSpecialChildren {
				obj.SetOwnerReferences(newObjOwners)
				remainingChildren = append(remainingChildren, obj)
				continue
			}

			sourceParents = append(sourceParents, ObjectsTree{
				UID:    string(obj.GetUID()),
				Kind:   obj.GetKind(),
				Object: obj.Object,
				Name:   obj.GetName(),
			})
			continue
		}
		remainingChildren = append(remainingChildren, obj)
	}
	return sourceParents, remainingChildren
}

func specialChildren(obj unstructured.Unstructured, specialParents map[string]map[string]unstructured.Unstructured) (bool, []v1.OwnerReference) {
	switch obj.GetKind() {
	case "Endpoints":
		service, ok := specialParents["Service"][fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName())]
		if ok {
			return true, []v1.OwnerReference{
				{
					APIVersion: service.GetAPIVersion(),
					Kind:       service.GetKind(),
					Name:       service.GetName(),
					UID:        service.GetUID(),
				},
			}
		}
	case "PersistentVolume":
		persistentVolumeClaim, ok := specialParents["PersistentVolumeClaim"][fmt.Sprintf("%s", obj.GetName())]
		if ok {
			return true, []v1.OwnerReference{
				{
					APIVersion: persistentVolumeClaim.GetAPIVersion(),
					Kind:       persistentVolumeClaim.GetKind(),
					Name:       persistentVolumeClaim.GetName(),
					UID:        persistentVolumeClaim.GetUID(),
				},
			}
		}
	case "PersistentVolumeClaim":
		pvcSplitName := strings.Split(obj.GetName(), "-")
		if len(pvcSplitName) == 1 {
			break
		}

		pvcNameWithoutIndex := strings.Join(pvcSplitName[:len(pvcSplitName)-1], "-")
		statefulSet := funk.Find(specialParents["StatefulSet"], func(statefulSet unstructured.Unstructured) bool {
			if obj.GetNamespace() == statefulSet.GetNamespace() &&
				strings.HasSuffix(pvcNameWithoutIndex, statefulSet.GetName()) {
				return true
			}
			return false
		})

		if statefulSet != nil {
			statefulSetUnstructured := statefulSet.(unstructured.Unstructured)
			return true, []v1.OwnerReference{
				{
					APIVersion: statefulSetUnstructured.GetAPIVersion(),
					Kind:       statefulSetUnstructured.GetKind(),
					Name:       statefulSetUnstructured.GetName(),
					UID:        statefulSetUnstructured.GetUID(),
				},
			}
		}
	}

	return false, nil
}

func getSpecialParents(objects []unstructured.Unstructured, specialParentsKind []string) map[string]map[string]unstructured.Unstructured {
	specialParents := make(map[string]map[string]unstructured.Unstructured)
	funk.ForEach(objects, func(obj unstructured.Unstructured) {
		if funk.ContainsString(specialParentsKind, obj.GetKind()) {
			if specialParents[obj.GetKind()] == nil {
				specialParents[obj.GetKind()] = make(map[string]unstructured.Unstructured)
			}

			switch obj.GetKind() {
			case "PersistentVolumeClaim":
				specialParents[obj.GetKind()][fmt.Sprintf("pvc-%s", obj.GetUID())] = obj
			default:
				specialParents[obj.GetKind()][fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName())] = obj
			}
		}
	})

	return specialParents
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
					Name:   obj.GetName(),
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
