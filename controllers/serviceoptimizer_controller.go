package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1beta1 "github.com/Nedopro2022/waofed/api/v1beta1"
)

// ServiceOptimizerReconciler reconciles a ServiceOptimizer object
type ServiceOptimizerReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	ControllerName string
}

//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.kubefed.io,resources=kubefedclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=types.kubefed.io,resources=federatedservices,verbs=get;list;watch
//+kubebuilder:rbac:groups=waofed.bitmedia.co.jp,resources=waofedconfigs,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceOptimizerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ControllerName = v1beta1.OperatorName + "-serviceoptimizer-controller"

	return ctrl.NewControllerManagedBy(mgr).
		For(newUnstructuredFederatedService()).
		Complete(r)
}

// Reconcile moves the current state of the cluster closer to the desired state.
func (r *ServiceOptimizerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx)
	lg.Info("Reconcile")

	// get WAOFedConfig
	wfc := &v1beta1.WAOFedConfig{}
	wfc.Name = v1beta1.WAOFedConfigName
	err := r.Get(ctx, client.ObjectKeyFromObject(wfc), wfc)
	if errors.IsNotFound(err) {
		lg.Info("no WAOFedConfig found, drop the request")
		return ctrl.Result{}, nil
	}
	if err != nil {
		lg.Error(err, fmt.Sprintf("unable to get WAOFedConfig %s", client.ObjectKeyFromObject(wfc)))
		return ctrl.Result{}, err
	}
	if wfc.Spec.LoadBalancing == nil {
		lg.Info("WAOFedConfig spec.loadbalancing is nil, drop the request")
		return ctrl.Result{}, nil
	}

	// get FederatedService
	fservice := newUnstructuredFederatedService()
	err = r.Get(ctx, req.NamespacedName, fservice)
	if errors.IsNotFound(err) {
		lg.Info("FederatedService is already deleted")
		return ctrl.Result{}, nil
	}
	if err != nil {
		lg.Error(err, "unable to get FederatedService")
		return ctrl.Result{}, nil
	}
	fsvc, err := convertToStructuredFederatedService(fservice)
	if err != nil {
		lg.Error(err, "unable to convert FederatedService")
		return ctrl.Result{}, err
	}

	// check if the FederatedService is using ServiceOptimizer
	skip := true
	if *wfc.Spec.LoadBalancing.Selector.Any {
		// check selector.any
		skip = false
	} else if _, ok := fsvc.GetAnnotations()[*wfc.Spec.LoadBalancing.Selector.HasAnnotation]; ok {
		// check ServiceOptimizer annotation exists in the FederatedService
		// currently the value is ignored
		skip = false
	}

	if skip {
		// Do nothing here for simplicity.
		// So you have to recreate a FederatedDeployment
		// to enable/disable ServiceOptimizer for the FederatedDeployment.
		// Future work: do some check here just like RSPOptimizerReconciler
		lg.Info("skip ServiceOptimizer")
		return ctrl.Result{}, nil
	}

	// TODO
	switch *wfc.Spec.LoadBalancing.Optimizer.Method {
	case v1beta1.ServiceOptimizerMethodRoundRobin:
	case v1beta1.ServiceOptimizerMethodWAO:
	default:
	}

	switch wfc.Spec.LoadBalancing.LoadBalancer.Type {
	case v1beta1.LoadBalancerTypeNone:
	case v1beta1.LoadBalancerTypeHAPRoxy:
	default:
	}

	return ctrl.Result{}, nil
}
