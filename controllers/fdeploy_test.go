package controllers

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

func Test_convertToStructuredFederatedDeployment(t *testing.T) {
	type args struct {
		in *unstructured.Unstructured
	}
	tests := []struct {
		name    string
		args    args
		want    *structuredFederatedDeployment
		wantErr bool
	}{
		{"normal",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedDeploymentObject.json")}},
			&structuredFederatedDeployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       federatedDeploymentGVK.Kind,
					APIVersion: federatedDeploymentGVK.GroupVersion().Identifier(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fdeploy-sample",
					Namespace: "default",
				},
				Spec: &structuredFederatedDeploymentSpec{
					Placement: &util.GenericPlacementFields{
						Clusters: []util.GenericClusterReference{
							{Name: "kind-waofed-1"}, {Name: "kind-waofed-2"}, {Name: "kind-waofed-3"}},
					},
					Template: &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "",
							Namespace: "",
						},
						Spec: appsv1.DeploymentSpec{
							Replicas: pointer.Int32(9),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"": "",
								},
							},
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "nginx",
											Image: "nginx:1.23.2",
										},
									},
								},
							},
						},
					},
				},
			},
			false,
		},
		{"no_placement",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedDeploymentObject_no_placement.json")}},
			&structuredFederatedDeployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       federatedDeploymentGVK.Kind,
					APIVersion: federatedDeploymentGVK.GroupVersion().Identifier(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fdeploy-sample",
					Namespace: "default",
				},
				Spec: &structuredFederatedDeploymentSpec{
					Template: &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "",
							Namespace: "",
						},
						Spec: appsv1.DeploymentSpec{
							Replicas: pointer.Int32(9),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"": "",
								},
							},
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "nginx",
											Image: "nginx:1.23.2",
										},
									},
								},
							},
						},
					},
				},
			},
			false,
		},
		{"no_template",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedDeploymentObject.json")}},
			&structuredFederatedDeployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       federatedDeploymentGVK.Kind,
					APIVersion: federatedDeploymentGVK.GroupVersion().Identifier(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fdeploy-sample",
					Namespace: "default",
				},
				Spec: &structuredFederatedDeploymentSpec{
					Placement: &util.GenericPlacementFields{
						Clusters: []util.GenericClusterReference{
							{Name: "kind-waofed-1"}, {Name: "kind-waofed-2"}, {Name: "kind-waofed-3"}},
					},
				},
			},
			false,
		},
		{"wrong GVK",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedDeploymentObject_wrong_GVK.json")}},
			&structuredFederatedDeployment{},
			true,
		},
		{"no_meta",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedDeploymentObject_no_meta.json")}},
			&structuredFederatedDeployment{},
			true,
		},
		{"no_spec",
			args{&unstructured.Unstructured{Object: helperLoadJSON(t, "testdata/unstructuredFederatedDeploymentObject_no_spec.json")}},
			&structuredFederatedDeployment{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToStructuredFederatedDeployment(tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToStructuredFederatedDeployment() error = %v, wantErr %v", err, tt.wantErr)
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
				compare("Replicas", got.Spec.Template.Spec.Replicas, tt.want.Spec.Template.Spec.Replicas)
				compare("Containers[0].Name", got.Spec.Template.Spec.Template.Spec.Containers[0].Name, tt.want.Spec.Template.Spec.Template.Spec.Containers[0].Name)
			}
			if !ok {
				t.Error("convertToStructuredFederatedDeployment()")
			}
		})
	}
}

func helperOpen(t *testing.T, name string) *os.File {
	f, err := os.Open(name)
	if err != nil {
		t.Error(err)
	}
	return f
}

func helperLoadJSON(t *testing.T, filename string) map[string]any {
	j := make(map[string]any)
	if err := json.NewDecoder(helperOpen(t, filename)).Decode(&j); err != nil {
		t.Error(err)
	}
	return j
}

func Test_sameOwner(t *testing.T) {
	type args struct {
		a metav1.OwnerReference
		b metav1.OwnerReference
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"same", args{
			a: metav1.OwnerReference{
				APIVersion:         "types.kubefed.io/v1beta1",
				Kind:               "FederatedDeployment",
				Name:               "fdeploy-sample",
				UID:                "db5574ab-3fbf-44ce-b8f0-4f231d333f6e",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
			},
			b: metav1.OwnerReference{
				APIVersion:         "types.kubefed.io/v1beta1",
				Kind:               "FederatedDeployment",
				Name:               "fdeploy-sample",
				UID:                "d3ea102a-f80b-4d0e-90a3-115924cd97a2",
				Controller:         nil,
				BlockOwnerDeletion: nil,
			},
		}, true},
		{"different", args{
			a: metav1.OwnerReference{
				APIVersion:         "types.kubefed.io/v1beta1",
				Kind:               "FederatedDeployment",
				Name:               "fdeploy-sample",
				UID:                "db5574ab-3fbf-44ce-b8f0-4f231d333f6e",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
			},
			b: metav1.OwnerReference{
				APIVersion:         "types.kubefed.io/v1beta1",
				Kind:               "FederatedDeployment",
				Name:               "fdeploy-sample2",
				UID:                "db5574ab-3fbf-44ce-b8f0-4f231d333f6e",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
			},
		}, false},
		{"invalid", args{
			a: metav1.OwnerReference{
				APIVersion:         "a/b/c",
				Kind:               "d",
				Name:               "e",
				UID:                "6a57571c-9b4b-44a4-bd3a-58fe139c51ce",
				Controller:         nil,
				BlockOwnerDeletion: nil,
			},
			b: metav1.OwnerReference{
				APIVersion:         "a/b/c",
				Kind:               "d",
				Name:               "e",
				UID:                "6a57571c-9b4b-44a4-bd3a-58fe139c51ce",
				Controller:         nil,
				BlockOwnerDeletion: nil,
			},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameOwner(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("sameOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

var (
	fdep1 = structuredFederatedDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       federatedDeploymentGVK.Kind,
			APIVersion: federatedDeploymentGVK.GroupVersion().Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fdeploy-sample",
			Namespace: "default",
		},
		Spec: &structuredFederatedDeploymentSpec{
			Placement: &util.GenericPlacementFields{
				Clusters: []util.GenericClusterReference{
					{Name: "kind-waofed-1"}, {Name: "kind-waofed-2"}, {Name: "kind-waofed-3"}},
			},
			Template: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "",
					Namespace: "",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: pointer.Int32(9),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"": "",
						},
					},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "nginx",
									Image: "nginx:1.23.2",
								},
							},
						},
					},
				},
			},
		},
	}
)

func Test_structuredFederatedDeploymentSetControllerReference(t *testing.T) {
	type args struct {
		owner      *structuredFederatedDeployment
		controlled metav1.Object
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"no owners", args{
			owner: &fdep1,
			controlled: &metav1.ObjectMeta{
				Name:            "hoge",
				Namespace:       "default",
				OwnerReferences: []metav1.OwnerReference{},
			},
		}, false},
		{"owned by self", args{
			owner: &fdep1,
			controlled: &metav1.ObjectMeta{
				Name:      "hoge",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "types.kubefed.io/v1beta1",
						Kind:               "FederatedDeployment",
						Name:               "fdeploy-sample",
						UID:                "",
						Controller:         pointer.Bool(false),
						BlockOwnerDeletion: pointer.Bool(false),
					},
				},
			},
		}, false},
		{"controlled by self", args{
			owner: &fdep1,
			controlled: &metav1.ObjectMeta{
				Name:      "hoge",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "types.kubefed.io/v1beta1",
						Kind:               "FederatedDeployment",
						Name:               "fdeploy-sample",
						UID:                "",
						Controller:         pointer.Bool(true),
						BlockOwnerDeletion: pointer.Bool(true),
					},
				},
			},
		}, false},
		{"controlled by other resource", args{
			owner: &fdep1,
			controlled: &metav1.ObjectMeta{
				Name:      "hoge",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "types.kubefed.io/v1beta1",
						Kind:               "FederatedDeployment",
						Name:               "fdeploy-sample2",
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
