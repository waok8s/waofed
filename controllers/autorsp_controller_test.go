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
				GroupVersionKind: federatedDeploymentGVK,
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fdeploy-sample",
					Namespace: "default",
				},
				Spec: struct {
					Template  appsv1.Deployment
					Placement util.GenericPlacementFields
				}{
					Placement: util.GenericPlacementFields{
						Clusters: []util.GenericClusterReference{
							{Name: "kind-waofed-1"}, {Name: "kind-waofed-2"}, {Name: "kind-waofed-3"}},
					},
					Template: appsv1.Deployment{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToStructuredFederatedDeployment(tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToStructuredFederatedDeployment() error = %v, wantErr %v", err, tt.wantErr)
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
			compare("GVK", got.GroupVersionKind, tt.want.GroupVersionKind)
			// metadata
			compare("Name", got.Name, tt.want.Name)
			compare("Namespace", got.Namespace, tt.want.Namespace)
			// placement
			compare("Clusters", got.Spec.Placement, tt.want.Spec.Placement)
			// template
			compare("Replicas", got.Spec.Template.Spec.Replicas, tt.want.Spec.Template.Spec.Replicas)
			compare("Containers[0].Name", got.Spec.Template.Spec.Template.Spec.Containers[0].Name, tt.want.Spec.Template.Spec.Template.Spec.Containers[0].Name)
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
