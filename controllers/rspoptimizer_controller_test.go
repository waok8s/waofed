package controllers

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	fedschedv1a1 "sigs.k8s.io/kubefed/pkg/apis/scheduling/v1alpha1"
)

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
			got, err := rspOptimizeFnRoundRobin(context.Background(), tt.args.clusters, nil, nil)
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
