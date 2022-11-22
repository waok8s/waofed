package controllers

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fedcorecommon "sigs.k8s.io/kubefed/pkg/apis/core/common"
	fedcorev1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	fedschedv1a1 "sigs.k8s.io/kubefed/pkg/apis/scheduling/v1alpha1"
)

func Test_listClusterNames(t *testing.T) {
	type args struct {
		cll         *fedcorev1b1.KubeFedClusterList
		statusReady bool
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"empty/true",
			args{
				cll: &fedcorev1b1.KubeFedClusterList{
					Items: []fedcorev1b1.KubeFedCluster{},
				},
				statusReady: true,
			},
			[]string{},
		},
		{"empty/false",
			args{
				cll: &fedcorev1b1.KubeFedClusterList{
					Items: []fedcorev1b1.KubeFedCluster{},
				},
				statusReady: false,
			},
			[]string{},
		},
		{"1cluster/ready/true",
			args{
				cll: &fedcorev1b1.KubeFedClusterList{Items: []fedcorev1b1.KubeFedCluster{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "c1"},
						Status: fedcorev1b1.KubeFedClusterStatus{Conditions: []fedcorev1b1.ClusterCondition{
							{Type: fedcorecommon.ClusterReady},
						}},
					},
				},
				},
				statusReady: true,
			},
			[]string{"c1"},
		},
		{"1cluster/non-ready/true",
			args{
				cll: &fedcorev1b1.KubeFedClusterList{Items: []fedcorev1b1.KubeFedCluster{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "c1"},
						Status: fedcorev1b1.KubeFedClusterStatus{Conditions: []fedcorev1b1.ClusterCondition{
							{Type: fedcorecommon.ClusterOffline},
						}},
					},
				},
				},
				statusReady: true,
			},
			[]string{},
		},
		{"1cluster/ready/false",
			args{
				cll: &fedcorev1b1.KubeFedClusterList{Items: []fedcorev1b1.KubeFedCluster{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "c1"},
						Status: fedcorev1b1.KubeFedClusterStatus{Conditions: []fedcorev1b1.ClusterCondition{
							{Type: fedcorecommon.ClusterReady},
						}},
					},
				},
				},
				statusReady: false,
			},
			[]string{"c1"},
		},
		{"1cluster/non-ready/false",
			args{
				cll: &fedcorev1b1.KubeFedClusterList{Items: []fedcorev1b1.KubeFedCluster{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "c1"},
						Status: fedcorev1b1.KubeFedClusterStatus{Conditions: []fedcorev1b1.ClusterCondition{
							{Type: fedcorecommon.ClusterOffline},
						}},
					},
				},
				},
				statusReady: false,
			},
			[]string{"c1"},
		},
		{"3clusters/true",
			args{
				cll: &fedcorev1b1.KubeFedClusterList{Items: []fedcorev1b1.KubeFedCluster{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "c1"},
						Status: fedcorev1b1.KubeFedClusterStatus{Conditions: []fedcorev1b1.ClusterCondition{
							{Type: fedcorecommon.ClusterReady},
						}},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "c2"},
						Status: fedcorev1b1.KubeFedClusterStatus{Conditions: []fedcorev1b1.ClusterCondition{
							{Type: fedcorecommon.ClusterOffline},
						}},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "c3"},
						Status: fedcorev1b1.KubeFedClusterStatus{Conditions: []fedcorev1b1.ClusterCondition{
							{Type: fedcorecommon.ClusterConfigMalformed},
						}},
					},
				},
				},
				statusReady: true,
			},
			[]string{"c1"},
		},
		{"3clusters/false",
			args{
				cll: &fedcorev1b1.KubeFedClusterList{Items: []fedcorev1b1.KubeFedCluster{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "c1"},
						Status: fedcorev1b1.KubeFedClusterStatus{Conditions: []fedcorev1b1.ClusterCondition{
							{Type: fedcorecommon.ClusterReady},
						}},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "c2"},
						Status: fedcorev1b1.KubeFedClusterStatus{Conditions: []fedcorev1b1.ClusterCondition{
							{Type: fedcorecommon.ClusterOffline},
						}},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "c3"},
						Status: fedcorev1b1.KubeFedClusterStatus{Conditions: []fedcorev1b1.ClusterCondition{
							{Type: fedcorecommon.ClusterConfigMalformed},
						}},
					},
				},
				},
				statusReady: false,
			},
			[]string{"c1", "c2", "c3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listClusterNames(tt.args.cll, tt.args.statusReady); cmp.Diff(got, tt.want) != "" {
				t.Errorf("listClusterNames() = %v, want %v, diff %s", got, tt.want, cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_optimizeFnRoundRobin(t *testing.T) {
	type args struct {
		clusters []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]fedschedv1a1.ClusterPreferences
		wantErr bool
	}{
		{"empty", args{[]string{}}, map[string]fedschedv1a1.ClusterPreferences{}, false},
		{"1cluster", args{[]string{"c1"}}, map[string]fedschedv1a1.ClusterPreferences{
			"c1": {
				MinReplicas: 0,
				MaxReplicas: nil,
				Weight:      1,
			},
		}, false},
		{"3clusters", args{[]string{"c1", "c2", "c3"}}, map[string]fedschedv1a1.ClusterPreferences{
			"c1": {
				MinReplicas: 0,
				MaxReplicas: nil,
				Weight:      1,
			},
			"c2": {
				MinReplicas: 0,
				MaxReplicas: nil,
				Weight:      1,
			},
			"c3": {
				MinReplicas: 0,
				MaxReplicas: nil,
				Weight:      1,
			},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := optimizeFnRoundRobin(tt.args.clusters)
			if (err != nil) != tt.wantErr {
				t.Errorf("optimizeFnRoundRobin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if cmp.Diff(got, tt.want) != "" {
				t.Errorf("optimizeFnRoundRobin() = %v, want %v, diff %s", got, tt.want, cmp.Diff(got, tt.want))
			}
		})
	}
}
