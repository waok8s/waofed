package controllers_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	v1beta1 "github.com/Nedopro2022/waofed/api/v1beta1"
	"github.com/Nedopro2022/waofed/controllers"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

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
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var (
	testNS        = "default"
	testKubeFedNS = "kube-federation-system"

	testWFC1 = v1beta1.WAOFedConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wfc1",
			Namespace: testNS,
		},
		Spec: v1beta1.WAOFedConfigSpec{
			KubeFedNamespace: testKubeFedNS,
			Scheduling: &v1beta1.SchedulingSettings{
				Selector: v1beta1.FederatedDeploymentSelector{
					Any:           false,
					HasAnnotation: v1beta1.DefaultRSPOptimizerAnnotation,
				},
				Optimizer: v1beta1.RSPOptimizerSettings{
					Method: v1beta1.DefaultRSPOptimizerAnnotation,
				},
			},
			LoadBalancing: &v1beta1.LoadBalancingSettings{},
		},
	}
)

// TODO: RSPOptimizer requires a KubeFed-enabled cluster to test. Might consider e2e tests.

var _ = Describe("WAOFedConfig controller", func() {

	var cncl context.CancelFunc

	BeforeEach(func() {

		ctx, cancel := context.WithCancel(context.Background())
		cncl = cancel

		var err error
		err = k8sClient.DeleteAllOf(ctx, &v1beta1.WAOFedConfig{}, client.InNamespace("")) // cluster-scoped
		Expect(err).NotTo(HaveOccurred())

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
		})
		Expect(err).NotTo(HaveOccurred())

		wfcReconciler := controllers.WAOFedConfigReconciler{
			Client: k8sClient,
			Scheme: scheme.Scheme,
		}
		err = wfcReconciler.SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		go func() {
			if err := mgr.Start(ctx); err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		cncl() // stop the mgr
		time.Sleep(100 * time.Millisecond)
	})

	It("should create WAOFedConfig", func() {

		wfc := testWFC1

		ctx := context.Background()

		err := k8sClient.Create(ctx, &wfc)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Create(ctx, &wfc)
		Expect(err).To(HaveOccurred())

	})
})
