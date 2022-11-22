package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	waofedv1beta1 "github.com/Nedopro2022/waofed/api/v1beta1"
)

// WAOFedConfigReconciler reconciles a WAOFedConfig object
type WAOFedConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=waofed.bitmedia.co.jp,resources=waofedconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=waofed.bitmedia.co.jp,resources=waofedconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=waofed.bitmedia.co.jp,resources=waofedconfigs/finalizers,verbs=update

// Reconcile moves the current state of the cluster closer to the desired state.
//
// WAOFedConfig is just a container that owns the configuration so nothing to do here.
func (r *WAOFedConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WAOFedConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&waofedv1beta1.WAOFedConfig{}).
		Complete(r)
}
