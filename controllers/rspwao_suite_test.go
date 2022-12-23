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
		type cps map[string]fedschedv1a1.ClusterPreferences
		const (
			c1 = "kind-waofed-test-0"
			c2 = "kind-waofed-test-1"
		)
		It("should be scheduled on", func() {
			// NOTE: use the first pattern at this time
			// [[0 1] [1 0]]
			testRSP(testWFCRSPWAO1, testNS, filepath.Join("testdata", "rspwao", "fdeploy15.yaml"),
				cps{c1: {Weight: 0}, c2: {Weight: 1}})
			// [[0 9] [1 8] [2 7] [3 6] [4 5] [5 4] [6 3] [7 2] [8 1] [9 0]]
			testRSP(testWFCRSPWAO1, testNS, filepath.Join("testdata", "rspwao", "fdeploy16.yaml"),
				cps{c1: {Weight: 0}, c2: {Weight: 9}})
		})
	})
})
