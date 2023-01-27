//go:build testOnExistingCluster

package controllers_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	fedcorev1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	fedschedv1a1 "sigs.k8s.io/kubefed/pkg/apis/scheduling/v1alpha1"

	v1beta1 "github.com/Nedopro2022/waofed/api/v1beta1"
	"github.com/Nedopro2022/waofed/controllers"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

var k8sDynamicClient dynamic.Interface

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    pointer.Bool(true),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = fedcorev1b1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = fedschedv1a1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = v1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// use the dynamic client for testing FederatedDeployment
	k8sDynamicClient, err = dynamic.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sDynamicClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func init() {
	SetDefaultEventuallyTimeout(20 * time.Second)
}

var (
	wait                  = func() { time.Sleep(100 * time.Millisecond) }
	testKubeFedNS         = "kube-federation-system"
	federatedNamespaceGVR = schema.GroupVersionResource{
		Group:    "types.kubefed.io",
		Version:  "v1beta1",
		Resource: "federatednamespaces",
	}
)

var (
	federatedDeploymentGVR = schema.GroupVersionResource{
		Group:    "types.kubefed.io",
		Version:  "v1beta1",
		Resource: "federateddeployments",
	}

	testNS = "default"

	testWFC1 = v1beta1.WAOFedConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: v1beta1.WAOFedConfigSpec{
			KubeFedNamespace: testKubeFedNS,
			Scheduling: &v1beta1.SchedulingSettings{
				Selector: &v1beta1.ResourceSelector{
					Any:           pointer.Bool(false),
					HasAnnotation: pointer.String(v1beta1.DefaultRSPOptimizerAnnotation),
				},
				Optimizer: &v1beta1.RSPOptimizerSettings{
					Method: (*v1beta1.RSPOptimizerMethod)(pointer.String(v1beta1.RSPOptimizerMethodRoundRobin)),
				},
			},
			LoadBalancing: &v1beta1.LoadBalancingSettings{},
		},
	}

	testWFC2 = v1beta1.WAOFedConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: v1beta1.WAOFedConfigSpec{
			KubeFedNamespace: testKubeFedNS,
			Scheduling: &v1beta1.SchedulingSettings{
				Selector: &v1beta1.ResourceSelector{
					Any: pointer.Bool(true),
				},
				Optimizer: &v1beta1.RSPOptimizerSettings{
					Method: (*v1beta1.RSPOptimizerMethod)(pointer.String(v1beta1.RSPOptimizerMethodRoundRobin)),
				},
			},
			LoadBalancing: &v1beta1.LoadBalancingSettings{},
		},
	}

	testWFC3 = v1beta1.WAOFedConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: v1beta1.WAOFedConfigSpec{
			KubeFedNamespace: testKubeFedNS,
			Scheduling:       nil,
			LoadBalancing:    &v1beta1.LoadBalancingSettings{},
		},
	}
)

var _ = Describe("RSPOptimizer controller", func() {
	var cncl context.CancelFunc

	BeforeEach(func() {
		ctx, cancel := context.WithCancel(context.Background())
		cncl = cancel

		var err error

		// create FederatedNamespace if not exists
		_, err = k8sDynamicClient.Resource(federatedNamespaceGVR).Namespace(testNS).Get(ctx, testNS, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			fns, _, _, err := helperLoadYAML(filepath.Join("testdata", "fns.yaml"))
			Expect(err).NotTo(HaveOccurred())
			_, err = k8sDynamicClient.Resource(federatedNamespaceGVR).Namespace(testNS).Create(ctx, fns, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
		}

		// delete all FederatedDeployment
		fdeployList, err := k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(testNS).List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		for _, fdeploy := range fdeployList.Items {
			err = k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(fdeploy.GetNamespace()).Delete(ctx, fdeploy.GetName(), metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
		for _, fdeploy := range fdeployList.Items {
			Eventually(func() error {
				_, err = k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(fdeploy.GetNamespace()).Get(ctx, fdeploy.GetName(), metav1.GetOptions{})
				return err
			}).ShouldNot(Succeed())
		}

		// delete all WAOFedConfig
		err = k8sClient.DeleteAllOf(ctx, &v1beta1.WAOFedConfig{}, client.InNamespace("")) // cluster-scoped
		Expect(err).NotTo(HaveOccurred())
		var wfc v1beta1.WAOFedConfig
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(&testWFC1), &wfc)
		}).ShouldNot(Succeed())

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
		})
		Expect(err).NotTo(HaveOccurred())

		rspOptimizerReconciler := controllers.RSPOptimizerReconciler{
			Client: k8sClient,
			Scheme: scheme.Scheme,
		}
		err = rspOptimizerReconciler.SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		go func() {
			if err := mgr.Start(ctx); err != nil {
				panic(err)
			}
		}()
		wait()
	})
	AfterEach(func() {
		cncl() // stop the mgr
		wait()
	})

	It("should not create RSP as no WAOFedConfig found", func() {

		ctx := context.Background()

		// create FederatedDeployment
		fdeploy, _, _, err := helperLoadYAML(filepath.Join("testdata", "fdeploy2.yaml"))
		Expect(err).NotTo(HaveOccurred())
		_, err = k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(testNS).Create(ctx, fdeploy, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// confirm RSP is NOT created
		rsp := &fedschedv1a1.ReplicaSchedulingPreference{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: fdeploy.GetNamespace(), Name: fdeploy.GetName()}, rsp)
		}).ShouldNot(Succeed())
	})

	It("should not create RSP as WAOFedConfig has no scheduling config", func() {

		wfc := testWFC3

		ctx := context.Background()

		// create WAOFedConfig
		err := k8sClient.Create(ctx, &wfc)
		Expect(err).NotTo(HaveOccurred())

		// create FederatedDeployment
		fdeploy, _, _, err := helperLoadYAML(filepath.Join("testdata", "fdeploy2.yaml"))
		Expect(err).NotTo(HaveOccurred())
		_, err = k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(testNS).Create(ctx, fdeploy, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// confirm RSP is NOT created
		rsp := &fedschedv1a1.ReplicaSchedulingPreference{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: fdeploy.GetNamespace(), Name: fdeploy.GetName()}, rsp)
		}).ShouldNot(Succeed())
	})

	It("should create, re-create and delete RSP", func() {

		wfc := testWFC1

		ctx := context.Background()

		// create WAOFedConfig
		err := k8sClient.Create(ctx, &wfc)
		Expect(err).NotTo(HaveOccurred())

		// create FederatedDeployment
		fdeploy, _, _, err := helperLoadYAML(filepath.Join("testdata", "fdeploy1.yaml"))
		Expect(err).NotTo(HaveOccurred())
		_, err = k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(testNS).Create(ctx, fdeploy, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// confirm RSP is also created
		rsp := &fedschedv1a1.ReplicaSchedulingPreference{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: fdeploy.GetNamespace(), Name: fdeploy.GetName()}, rsp)
		}).Should(Succeed())

		// delete the RSP and confirm the re-creation
		err = k8sClient.Delete(ctx, rsp)
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: fdeploy.GetNamespace(), Name: fdeploy.GetName()}, rsp)
		}).Should(Succeed())

		// delete FederatedDeployment
		err = k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(fdeploy.GetNamespace()).Delete(ctx, fdeploy.GetName(), metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		// confirm RSP is also deleted
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: fdeploy.GetNamespace(), Name: fdeploy.GetName()}, rsp)
		}).ShouldNot(Succeed())
	})

	It("should delete RSP when annotation deleted from FederatedDeployment", func() {

		wfc := testWFC1

		ctx := context.Background()

		// create WAOFedConfig
		err := k8sClient.Create(ctx, &wfc)
		Expect(err).NotTo(HaveOccurred())

		// create FederatedDeployment
		fdeploy, _, _, err := helperLoadYAML(filepath.Join("testdata", "fdeploy1.yaml"))
		Expect(err).NotTo(HaveOccurred())
		_, err = k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(testNS).Create(ctx, fdeploy, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// confirm RSP is also created
		rsp := &fedschedv1a1.ReplicaSchedulingPreference{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: fdeploy.GetNamespace(), Name: fdeploy.GetName()}, rsp)
		}).Should(Succeed())

		// delete annotation from FederatedDeployment
		annotationInPatchFormat := strings.ReplaceAll(*wfc.Spec.Scheduling.Selector.HasAnnotation, "/", "~1")
		patch := []byte(`[{"op": "remove", "path": "/metadata/annotations/` + annotationInPatchFormat + `"}]`)
		fdeploy, err = k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(fdeploy.GetNamespace()).Patch(ctx, fdeploy.GetName(), types.JSONPatchType, patch, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		// confirm RSP is deleted
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: fdeploy.GetNamespace(), Name: fdeploy.GetName()}, rsp)
		}).ShouldNot(Succeed())
	})

	It("should not create RSP as no annotations specified in FederatedDeployment", func() {

		wfc := testWFC1

		ctx := context.Background()

		// create WAOFedConfig
		err := k8sClient.Create(ctx, &wfc)
		Expect(err).NotTo(HaveOccurred())

		// create FederatedDeployment
		fdeploy, _, _, err := helperLoadYAML(filepath.Join("testdata", "fdeploy2.yaml"))
		Expect(err).NotTo(HaveOccurred())
		_, err = k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(testNS).Create(ctx, fdeploy, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// confirm RSP is NOT created
		rsp := &fedschedv1a1.ReplicaSchedulingPreference{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: fdeploy.GetNamespace(), Name: fdeploy.GetName()}, rsp)
		}).ShouldNot(Succeed())
	})

	It("should create RSP as selector.any=true", func() {

		wfc := testWFC2

		ctx := context.Background()

		// create WAOFedConfig
		err := k8sClient.Create(ctx, &wfc)
		Expect(err).NotTo(HaveOccurred())

		// create FederatedDeployment
		fdeploy, _, _, err := helperLoadYAML(filepath.Join("testdata", "fdeploy2.yaml"))
		Expect(err).NotTo(HaveOccurred())
		_, err = k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(testNS).Create(ctx, fdeploy, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// confirm RSP is also created
		rsp := &fedschedv1a1.ReplicaSchedulingPreference{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: fdeploy.GetNamespace(), Name: fdeploy.GetName()}, rsp)
		}).Should(Succeed())
	})

	Context("schedule on clusters", func() {
		wantX := map[string]fedschedv1a1.ClusterPreferences{}
		_ = wantX
		want0 := map[string]fedschedv1a1.ClusterPreferences{
			"kind-waofed-test-0": {MinReplicas: 0, MaxReplicas: nil, Weight: 1},
		}
		_ = want0
		want1 := map[string]fedschedv1a1.ClusterPreferences{
			"kind-waofed-test-1": {MinReplicas: 0, MaxReplicas: nil, Weight: 1},
		}
		_ = want1
		want01 := map[string]fedschedv1a1.ClusterPreferences{
			"kind-waofed-test-0": {MinReplicas: 0, MaxReplicas: nil, Weight: 1},
			"kind-waofed-test-1": {MinReplicas: 0, MaxReplicas: nil, Weight: 1},
		}
		_ = want01
		It("should be scheduled on cluster0", func() {
			testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy3.yaml"), want0)
			testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy4.yaml"), want0)
		})
		It("should be scheduled on cluster1", func() {
			testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy5.yaml"), want1)
			testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy6.yaml"), want1)
		})
		It("should be scheduled on cluster0 and cluster1", func() {
			testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy7.yaml"), want01)
			testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy8.yaml"), want01)
			testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy9.yaml"), want01)
		})
		It("should not be scheduled on any cluster", func() {
			testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy10.yaml"), wantX)
			testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy11.yaml"), wantX)
			testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy12.yaml"), wantX)
			testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy13.yaml"), wantX)
			// NOTE: uncovered edge case, see readme for details
			// testRSP(testWFC2, testNS, filepath.Join("testdata", "fdeploy14.yaml"), wantX)
		})
	})
})

func testRSP(wfc v1beta1.WAOFedConfig, fdeployNS, fdeployFile string, want map[string]fedschedv1a1.ClusterPreferences) {
	ctx := context.Background()

	// create WAOFedConfig if not exists
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&wfc), &wfc); errors.IsNotFound(err) {
		err := k8sClient.Create(ctx, &wfc)
		Expect(err).NotTo(HaveOccurred())
	}

	// create FederatedDeployment
	fdeploy, _, _, err := helperLoadYAML(fdeployFile)
	Expect(err).NotTo(HaveOccurred())
	_, err = k8sDynamicClient.Resource(federatedDeploymentGVR).Namespace(fdeployNS).Create(ctx, fdeploy, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	// confirm RSP is also created
	rsp := &fedschedv1a1.ReplicaSchedulingPreference{}
	Eventually(func() error {
		return k8sClient.Get(ctx, client.ObjectKey{Namespace: fdeploy.GetNamespace(), Name: fdeploy.GetName()}, rsp)
	}).Should(Succeed())

	// check RSP
	if rsp.Spec.Clusters == nil {
		// both want == nil or want == map[string]fedschedv1a1.ClusterPreferences{} are ok
		Expect(want == nil || cmp.Diff(want, map[string]fedschedv1a1.ClusterPreferences{}) == "").Should(BeTrue())
	} else {
		Expect(cmp.Diff(want, rsp.Spec.Clusters)).Should(BeEmpty())
	}
}

func helperLoadYAML(name string) (*unstructured.Unstructured, runtime.Object, *schema.GroupVersionKind, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, nil, nil, err
	}
	p, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, nil, err
	}
	obj := &unstructured.Unstructured{}
	ro, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(p, nil, obj)
	if err != nil {
		return nil, nil, nil, err
	}
	return obj, ro, gvk, err
}

var (
	federatedServiceGVR = schema.GroupVersionResource{
		Group:    "types.kubefed.io",
		Version:  "v1beta1",
		Resource: "federatedservices",
	}
)

var _ = Describe("SLPOptimizer controller", func() {
	var cncl context.CancelFunc

	BeforeEach(func() {

		ctx, cancel := context.WithCancel(context.Background())
		cncl = cancel

		var err error

		// create FederatedNamespace if not exists
		_, err = k8sDynamicClient.Resource(federatedNamespaceGVR).Namespace(testNS).Get(ctx, testNS, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			fns, _, _, err := helperLoadYAML(filepath.Join("testdata", "fns.yaml"))
			Expect(err).NotTo(HaveOccurred())
			_, err = k8sDynamicClient.Resource(federatedNamespaceGVR).Namespace(testNS).Create(ctx, fns, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
		}

		// delete all FederatedService
		fsvcList, err := k8sDynamicClient.Resource(federatedServiceGVR).Namespace(testNS).List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		for _, fsvc := range fsvcList.Items {
			err = k8sDynamicClient.Resource(federatedServiceGVR).Namespace(fsvc.GetNamespace()).Delete(ctx, fsvc.GetName(), metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
		for _, fsvc := range fsvcList.Items {
			Eventually(func() error {
				_, err = k8sDynamicClient.Resource(federatedServiceGVR).Namespace(fsvc.GetNamespace()).Get(ctx, fsvc.GetName(), metav1.GetOptions{})
				return err
			}).ShouldNot(Succeed())
		}

		// delete all WAOFedConfig
		err = k8sClient.DeleteAllOf(ctx, &v1beta1.WAOFedConfig{}, client.InNamespace("")) // cluster-scoped
		Expect(err).NotTo(HaveOccurred())
		var wfc v1beta1.WAOFedConfig
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(&testWFC1), &wfc)
		}).ShouldNot(Succeed())

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
		})
		Expect(err).NotTo(HaveOccurred())

		slpOptimizerReconciler := controllers.SLPOptimizerReconciler{
			Client: k8sClient,
			Scheme: scheme.Scheme,
		}
		err = slpOptimizerReconciler.SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		go func() {
			if err := mgr.Start(ctx); err != nil {
				panic(err)
			}
		}()
		wait()
	})
	AfterEach(func() {
		cncl() // stop the mgr
		wait()
	})

	It("should not do anything as no WAOFedConfig found", func() {

		ctx := context.Background()

		// create FederatedService
		fsvc, _, _, err := helperLoadYAML(filepath.Join("testdata", "fsvc2.yaml"))
		Expect(err).NotTo(HaveOccurred())
		_, err = k8sDynamicClient.Resource(federatedServiceGVR).Namespace(testNS).Create(ctx, fsvc, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// TODO: do some confirmations
	})

})
