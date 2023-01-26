// Package controllers provides controllers
//
// NOTE: structuredFederatedDeployment has metav1.TypeMeta field,
// which cause Kubebuilder to see it as an API and try to generate a CRD manifest,
// but actually it is not an API, so set kubebuilder:skip to avoid this behavior.
// +kubebuilder:skip
package controllers

import (
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	fedctrlutil "sigs.k8s.io/kubefed/pkg/controller/util"
)

var federatedDeploymentGVK = schema.GroupVersionKind{
	Group:   "types.kubefed.io",
	Kind:    "FederatedDeployment",
	Version: "v1beta1",
}

func newUnstructuredFederatedDeployment() *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(federatedDeploymentGVK)
	return u
}

type structuredFederatedDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              *structuredFederatedDeploymentSpec `json:"spec,omitempty"`
}

type structuredFederatedDeploymentSpec struct {
	Template  *appsv1.Deployment                  `json:"template,omitempty"`
	Placement *fedctrlutil.GenericPlacementFields `json:"placement,omitempty"`
}

func convertToStructuredFederatedDeployment(in *unstructured.Unstructured) (*structuredFederatedDeployment, error) {
	var out structuredFederatedDeployment
	out.Spec = &structuredFederatedDeploymentSpec{}

	if in.GroupVersionKind() != federatedDeploymentGVK {
		return nil, fmt.Errorf("wrong GVK: %v", in.GroupVersionKind())
	}
	out.TypeMeta = metav1.TypeMeta{
		Kind:       federatedDeploymentGVK.Kind,
		APIVersion: federatedDeploymentGVK.GroupVersion().Identifier(),
	}

	objMeta, err := convertUnstructuredFieldToObject[*metav1.ObjectMeta]("metadata", in.Object)
	if err != nil {
		return nil, err
	}
	out.ObjectMeta = *objMeta

	v, ok := in.Object["spec"]
	if !ok {
		return nil, fmt.Errorf("could not get %s", "spec")
	}
	spec, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("could not encode %s", "spec")
	}

	objPlacement, err := convertUnstructuredFieldToObject[*fedctrlutil.GenericPlacementFields]("placement", spec)
	if err == nil {
		out.Spec.Placement = objPlacement
	}

	objDeployment, err := convertUnstructuredFieldToObject[*appsv1.Deployment]("template", spec)
	if err == nil {
		out.Spec.Template = objDeployment
	}

	return &out, nil
}

func convertUnstructuredFieldToObject[T any](fieldName string, unstructuredObj map[string]any) (T, error) {
	var obj T
	v, ok := unstructuredObj[fieldName]
	if !ok {
		return obj, fmt.Errorf("could not get %s", fieldName)
	}

	// NOTE: Type assertion doesn't work, need to convert via JSON.
	//
	// obj, ok = v.(T)
	// if !ok { // always false
	// 	return obj, fmt.Errorf("bad type assertion")
	// }

	p, err := json.Marshal(&v)
	if err != nil {
		return obj, fmt.Errorf("could not encode %s: %v", fieldName, err)
	}
	if err := json.Unmarshal(p, &obj); err != nil {
		return obj, fmt.Errorf("could not decode %s: %v", fieldName, err)
	}

	// DEBUG
	// fmt.Printf("convertUnstructuredFieldToObject: %s\njson:\n%s\nobj:%#v\n", fieldName, p, obj)

	return obj, nil
}

func (r *structuredFederatedDeployment) setControllerReference(controlled metav1.Object) error {
	newRef := metav1.OwnerReference{
		APIVersion:         r.APIVersion,
		Kind:               r.Kind,
		Name:               r.Name,
		UID:                r.UID,
		Controller:         pointer.Bool(true),
		BlockOwnerDeletion: pointer.BoolPtr(true),
	}

	// return error if controlled by other resource
	if curRef := metav1.GetControllerOf(controlled); curRef != nil && !(sameOwner(newRef, *curRef)) {
		return fmt.Errorf("already owned by GVK=%s.%s Name=%s", curRef.Kind, curRef.APIVersion, curRef.Name)
	}

	// append the OwnerReference or replace the old one with it
	refs := controlled.GetOwnerReferences()
	idx := -1
	for i, r := range refs {
		if sameOwner(newRef, r) {
			idx = i
		}
	}
	if idx == -1 {
		refs = append(refs, newRef)
	} else {
		refs[idx] = newRef
	}
	controlled.SetOwnerReferences(refs)

	return nil
}

func sameOwner(a, b metav1.OwnerReference) bool {
	aa, errA := schema.ParseGroupVersion(a.APIVersion)
	bb, errB := schema.ParseGroupVersion(b.APIVersion)
	if errA != nil || errB != nil {
		return false
	}
	return (aa.Group == bb.Group) && (a.Kind == b.Kind) && (a.Name == b.Name)
}
