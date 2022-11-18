package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1beta1 "github.com/Nedopro2022/waofed/api/v1beta1"
)

const (
	ControllerName = v1beta1.OperatorName + "-autorsp-controller"
	AnnotAutoRSP   = "wao-autorsp"
)

// AutoRSPReconciler reconciles a AutoRSP object
type AutoRSPReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=types.kubefed.io,resources=federateddeployments,verbs=get;list;watch;
//+kubebuilder:rbac:groups=scheduling.kubefed.io,resources=replicaschedulingpreferences,verbs=get;list;watch;create;update;patch;delete

// Reconcile moves the current state of the cluster closer to the desired state.
func (r *AutoRSPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx)
	lg.Info("Reconcile")

	// check AutoRSP annotation exists in FederatedDeployment

	fdep := NewFederatedDeploymentInUnstructured()
	err := r.Get(ctx, req.NamespacedName, fdep)
	if errors.IsNotFound(err) {
		lg.Info("FederatedDeployment is already deleted")
		return ctrl.Result{}, nil
	}
	if err != nil {
		lg.Error(err, "unable to get FederatedDeployment")
		return ctrl.Result{}, err
	}
	_, ok := fdep.GetAnnotations()[AnnotAutoRSP]
	if !ok {
		lg.Info("FederatedDeployment doesn't have AutoRSP annotation")
		return ctrl.Result{}, nil
	}

	// TODO: create RSP

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AutoRSPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(NewFederatedDeploymentInUnstructured()).
		Complete(r)
}

func NewFederatedDeploymentInUnstructured() *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "types.kubefed.io",
		Kind:    "FederatedDeployment",
		Version: "v1beta1",
	})
	return u
}
