//go:build testOnExistingClusterRSPWAO

package controllers_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	fedschedv1a1 "sigs.k8s.io/kubefed/pkg/apis/scheduling/v1alpha1"

	v1beta1 "github.com/Nedopro2022/waofed/api/v1beta1"
)

var (
	testWFCRSPWAO1 = v1beta1.WAOFedConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: v1beta1.WAOFedConfigSpec{
			KubeFedNamespace: testKubeFedNS,
			Scheduling: &v1beta1.SchedulingSettings{
				Selector: &v1beta1.FederatedDeploymentSelector{
					Any: pointer.Bool(true),
				},
				Optimizer: &v1beta1.RSPOptimizerSettings{
					Method: (*v1beta1.RSPOptimizerMethod)(pointer.String(v1beta1.RSPOptimizerMethodWAO)),
					WAOEstimators: map[string]*v1beta1.WAOEstimatorSetting{
						"kind-waofed-test-0": {
							Endpoint:  "http://localhost:5657",
							Namespace: "default",
							Name:      "default",
						},
						"kind-waofed-test-1": {
							Endpoint:  "http://localhost:5658",
							Namespace: "default",
							Name:      "default",
						},
					},
				},
			},
			LoadBalancing: &v1beta1.LoadBalancingSettings{},
		},
	}
)

var _ = Describe("WAOFedConfig controller (RSPWAO)", func() {

	BeforeEach(beforeEachFn)
	AfterEach(afterEachFn)

	Context("schedule on clusters (RSPWAO)", func() {
		wantX := map[string]fedschedv1a1.ClusterPreferences{}
		_ = wantX
		want10 := map[string]fedschedv1a1.ClusterPreferences{
			"kind-waofed-test-0": {MinReplicas: 0, MaxReplicas: nil, Weight: 1},
			"kind-waofed-test-1": {MinReplicas: 0, MaxReplicas: nil, Weight: 0},
		}
		_ = want10
		want01 := map[string]fedschedv1a1.ClusterPreferences{
			"kind-waofed-test-0": {MinReplicas: 0, MaxReplicas: nil, Weight: 0},
			"kind-waofed-test-1": {MinReplicas: 0, MaxReplicas: nil, Weight: 1},
		}
		_ = want01
		want11 := map[string]fedschedv1a1.ClusterPreferences{
			"kind-waofed-test-0": {MinReplicas: 0, MaxReplicas: nil, Weight: 1},
			"kind-waofed-test-1": {MinReplicas: 0, MaxReplicas: nil, Weight: 1},
		}
		_ = want11
		It("should be scheduled on cluster0", func() {
		})
		It("should be scheduled on cluster1", func() {
			testRSP(testWFCRSPWAO1, testNS, filepath.Join("testdata", "rspwao", "fdeploy15.yaml"), want01)
		})
		It("should be scheduled on cluster0 and cluster1", func() {
		})
		It("should not be scheduled on any cluster", func() {
		})
	})
})
