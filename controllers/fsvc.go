// Package controllers provides controllers
//
// NOTE: structuredFederatedService has metav1.TypeMeta field,
// which cause Kubebuilder to see it as an API and try to generate a CRD manifest,
// but actually it is not an API, so set kubebuilder:skip to avoid this behavior.
// +kubebuilder:skip
package controllers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	fedctrlutil "sigs.k8s.io/kubefed/pkg/controller/util"
)

var federatedServiceGVK = schema.GroupVersionKind{
	Group:   "types.kubefed.io",
	Kind:    "FederatedService",
	Version: "v1beta1",
}

func newUnstructuredFederatedService() *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(federatedServiceGVK)
	return u
}

type structuredFederatedService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              *structuredFederatedServiceSpec `json:"spec,omitempty"`
}

type structuredFederatedServiceSpec struct {
	Template  *corev1.Service                     `json:"template,omitempty"`
	Placement *fedctrlutil.GenericPlacementFields `json:"placement,omitempty"`
}

func convertToStructuredFederatedService(in *unstructured.Unstructured) (*structuredFederatedService, error) {
	var out structuredFederatedService
	out.Spec = &structuredFederatedServiceSpec{}

	if in.GroupVersionKind() != federatedServiceGVK {
		return nil, fmt.Errorf("wrong GVK: %v", in.GroupVersionKind())
	}
	out.TypeMeta = metav1.TypeMeta{
		Kind:       federatedServiceGVK.Kind,
		APIVersion: federatedServiceGVK.GroupVersion().Identifier(),
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

	objService, err := convertUnstructuredFieldToObject[*corev1.Service]("template", spec)
	if err == nil {
		out.Spec.Template = objService
	}

	return &out, nil
}

func (r *structuredFederatedService) setControllerReference(controlled metav1.Object) error {
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
