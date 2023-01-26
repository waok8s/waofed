package controllers

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

func Test_convertToStructuredFederatedService(t *testing.T) {
	type args struct {
		in *unstructured.Unstructured
	}
	tests := []struct {
		name    string
		args    args
		want    *structuredFederatedService
		wantErr bool
	}{
		{"normal",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedServiceObject.json")}},
			&structuredFederatedService{
				TypeMeta: metav1.TypeMeta{
					Kind:       federatedServiceGVK.Kind,
					APIVersion: federatedServiceGVK.GroupVersion().Identifier(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsvc-sample",
					Namespace: "default",
				},
				Spec: &structuredFederatedServiceSpec{
					Placement: &util.GenericPlacementFields{
						Clusters: []util.GenericClusterReference{
							{Name: "kind-waofed-1"}, {Name: "kind-waofed-2"}, {Name: "kind-waofed-3"}},
					},
					Template: &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "",
							Namespace: "",
						},
						Spec: corev1.ServiceSpec{
							Selector: map[string]string{
								"app": "nginx",
							},
							Ports: []corev1.ServicePort{
								{Name: "http", Port: 80},
							},
							Type: corev1.ServiceTypeLoadBalancer,
						},
					},
				},
			},
			false,
		},
		{"no_placement",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedServiceObject_no_placement.json")}},
			&structuredFederatedService{
				TypeMeta: metav1.TypeMeta{
					Kind:       federatedServiceGVK.Kind,
					APIVersion: federatedServiceGVK.GroupVersion().Identifier(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsvc-sample",
					Namespace: "default",
				},
				Spec: &structuredFederatedServiceSpec{
					Template: &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "",
							Namespace: "",
						},
						Spec: corev1.ServiceSpec{
							Selector: map[string]string{
								"app": "nginx",
							},
							Ports: []corev1.ServicePort{
								{Name: "http", Port: 80},
							},
							Type: corev1.ServiceTypeLoadBalancer,
						},
					},
				},
			},
			false,
		},
		{"no_template",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedServiceObject_no_template.json")}},
			&structuredFederatedService{
				TypeMeta: metav1.TypeMeta{
					Kind:       federatedServiceGVK.Kind,
					APIVersion: federatedServiceGVK.GroupVersion().Identifier(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsvc-sample",
					Namespace: "default",
				},
				Spec: &structuredFederatedServiceSpec{
					Placement: &util.GenericPlacementFields{
						Clusters: []util.GenericClusterReference{
							{Name: "kind-waofed-1"}, {Name: "kind-waofed-2"}, {Name: "kind-waofed-3"}},
					},
				},
			},
			false,
		},
		{"wrong GVK",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedServiceObject_wrong_GVK.json")}},
			&structuredFederatedService{},
			true,
		},
		{"no_meta",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedServiceObject_no_meta.json")}},
			&structuredFederatedService{},
			true,
		},
		{"no_spec",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedServiceObject_no_spec.json")}},
			&structuredFederatedService{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToStructuredFederatedService(tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToStructuredFederatedService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			ok := true
			compare := func(name string, x any, y any) {
				if !cmp.Equal(x, y) {
					t.Logf("NE: %s got=%v want=%v", name, x, y)
					ok = false
				} else {
					t.Logf("EQ: %s", name)
				}
			}
			// GVK
			compare("APIVersion", got.APIVersion, tt.want.APIVersion)
			compare("Kind", got.Kind, tt.want.Kind)
			// metadata
			compare("Name", got.Name, tt.want.Name)
			compare("Namespace", got.Namespace, tt.want.Namespace)
			// placement
			compare("Clusters", got.Spec.Placement, tt.want.Spec.Placement)
			// template
			if tt.want.Spec.Template != nil {
				compare("Type", got.Spec.Template.Spec.Type, tt.want.Spec.Template.Spec.Type)
				compare("Selector", got.Spec.Template.Spec.Selector, tt.want.Spec.Template.Spec.Selector)
				compare("Ports[0].Port", got.Spec.Template.Spec.Ports[0].Port, tt.want.Spec.Template.Spec.Ports[0].Port)
			}
			if !ok {
				t.Error("convertToStructuredFederatedService()")
			}
		})
	}
}

var (
	fsvc1 = structuredFederatedService{
		TypeMeta: metav1.TypeMeta{
			Kind:       federatedServiceGVK.Kind,
			APIVersion: federatedServiceGVK.GroupVersion().Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fsvc-sample",
			Namespace: "default",
		},
		Spec: &structuredFederatedServiceSpec{
			Placement: &util.GenericPlacementFields{
				Clusters: []util.GenericClusterReference{
					{Name: "kind-waofed-1"}, {Name: "kind-waofed-2"}, {Name: "kind-waofed-3"}},
			},
			Template: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "",
					Namespace: "",
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						"app": "nginx",
					},
					Ports: []corev1.ServicePort{
						{Name: "http", Port: 80},
					},
					Type: corev1.ServiceTypeLoadBalancer,
				},
			},
		},
	}
)

func Test_structuredFederatedServiceSetControllerReference(t *testing.T) {
	type args struct {
		owner      *structuredFederatedService
		controlled metav1.Object
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"no owners", args{
			owner: &fsvc1,
			controlled: &metav1.ObjectMeta{
				Name:            "hoge",
				Namespace:       "default",
				OwnerReferences: []metav1.OwnerReference{},
			},
		}, false},
		{"owned by self", args{
			owner: &fsvc1,
			controlled: &metav1.ObjectMeta{
				Name:      "hoge",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "types.kubefed.io/v1beta1",
						Kind:               "FederatedService",
						Name:               "fsvc-sample",
						UID:                "",
						Controller:         pointer.Bool(false),
						BlockOwnerDeletion: pointer.Bool(false),
					},
				},
			},
		}, false},
		{"controlled by self", args{
			owner: &fsvc1,
			controlled: &metav1.ObjectMeta{
				Name:      "hoge",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "types.kubefed.io/v1beta1",
						Kind:               "FederatedService",
						Name:               "fsvc-sample",
						UID:                "",
						Controller:         pointer.Bool(true),
						BlockOwnerDeletion: pointer.Bool(true),
					},
				},
			},
		}, false},
		{"controlled by other resource", args{
			owner: &fsvc1,
			controlled: &metav1.ObjectMeta{
				Name:      "hoge",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "types.kubefed.io/v1beta1",
						Kind:               "FederatedService",
						Name:               "fsvc-sample2",
						UID:                "",
						Controller:         pointer.Bool(true),
						BlockOwnerDeletion: pointer.Bool(true),
					},
				},
			},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.args.owner.setControllerReference(tt.args.controlled); (err != nil) != tt.wantErr {
				t.Errorf("setControllerReference() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
