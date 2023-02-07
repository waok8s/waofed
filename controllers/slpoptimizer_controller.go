package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	fedcorev1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"

	v1beta1 "github.com/Nedopro2022/waofed/api/v1beta1"
)

// SLPOptimizerReconciler reconciles a SLPOptimizer object
type SLPOptimizerReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	ControllerName string
}

//+kubebuilder:rbac:groups=core.kubefed.io,resources=kubefedclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=types.kubefed.io,resources=federatedservices,verbs=get;list;watch
//+kubebuilder:rbac:groups=waofed.bitmedia.co.jp,resources=waofedconfigs,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *SLPOptimizerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ControllerName = v1beta1.OperatorName + "-SLPOptimizer-controller"

	return ctrl.NewControllerManagedBy(mgr).
		For(newUnstructuredFederatedService()).
		Owns(&v1beta1.ServiceLoadbalancingPreference{}).
		Complete(r)
}

// Reconcile moves the current state of the cluster closer to the desired state.
func (r *SLPOptimizerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	// reconcile SLP
	if err := r.reconcileLSP(ctx, fsvc, wfc); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SLPOptimizerReconciler) reconcileLSP(
	ctx context.Context, fsvc *structuredFederatedService, wfc *v1beta1.WAOFedConfig,
) error {
	lg := log.FromContext(ctx)
	lg.Info("reconcileRSP")

	skip := true
	if *wfc.Spec.LoadBalancing.Selector.Any {
		// check selector.any
		skip = false
	} else if _, ok := fsvc.GetAnnotations()[*wfc.Spec.LoadBalancing.Selector.HasAnnotation]; ok {
		// check SLPOptimizer annotation exists in the FederatedService
		// currently the value is ignored
		skip = false
	}

	if skip {
		// delete the associated SLP if no annotation in the FederatedService
		// Ref. RSPOptimizerReconciler.reconcileRSP (same implementation)
		lg.Info("FederatedService doesn't have SLPOptimizer annotation")
		slp := &v1beta1.ServiceLoadbalancingPreference{}
		slp.SetNamespace(fsvc.Namespace)
		slp.SetName(fsvc.Name)
		err := r.Get(ctx, client.ObjectKeyFromObject(slp), slp)
		if errors.IsNotFound(err) {
			lg.Info("SLP is already deleted")
			return nil
		}
		if err != nil {
			lg.Error(err, "unable to get SLP")
			return err
		}
		ctrlRef := metav1.GetControllerOf(slp)
		compareRef := &metav1.OwnerReference{
			APIVersion: fsvc.APIVersion,
			Kind:       fsvc.Kind,
			Name:       fsvc.Name,
		}
		if ctrlRef != nil && sameOwner(*ctrlRef, *compareRef) {
			lg.Info("SLP having a Controller OwnerReference implies the SLP was created by SLPOptimizer, so delete it")
			err := r.Delete(ctx, slp)
			if errors.IsNotFound(err) {
				lg.Info("SLP is already deleted")
				return nil
			}
			if err != nil {
				lg.Error(err, "unable to delete LSP")
				return err
			}
		}
		return nil
	} else {
		// apply SLP if !skip
		// Ref. RSPOptimizerReconciler.reconcileRSP (same implementation)
		slp := &v1beta1.ServiceLoadbalancingPreference{}
		slp.SetNamespace(fsvc.Namespace)
		slp.SetName(fsvc.Name)
		op, err := ctrl.CreateOrUpdate(ctx, r.Client, slp, func() error {
			slp.Labels = map[string]string{
				"app.kubernetes.io/created-by": r.ControllerName,
			}
			slp.Spec = v1beta1.ServiceLoadbalancingPreferenceSpec{
				Clusters: nil,
			}
			lg.Info("optimize cluster weights", "method", wfc.Spec.LoadBalancing.Optimizer.Method)
			clusters, err := r.optimizeClusterWeights(ctx, fsvc, wfc)
			if err != nil {
				return err
			}
			slp.Spec.Clusters = clusters
			if err := fsvc.setControllerReference(slp); err != nil {
				return err
			}
			return err
		})
		if err != nil {
			lg.Error(err, "unable to create or update SLP")
		}
		lg.Info("SLP operated", "op", op)
	}

	return nil
}

func (r *SLPOptimizerReconciler) optimizeClusterWeights(
	ctx context.Context, fsvc *structuredFederatedService, wfc *v1beta1.WAOFedConfig,
) (map[string]v1beta1.ClusterPreferences, error) {
	lg := log.FromContext(ctx)
	lg.Info("optimizeClusterWeights", "wfc", wfc, "fsvc", fsvc)

	// Ref. RSPOptimizerReconciler.optimizeClusterWeights (same implementation)

	if fsvc.Spec.Placement == nil || (fsvc.Spec.Placement.Clusters == nil && fsvc.Spec.Placement.ClusterSelector == nil) {
		lg.Info("no loadbalancing as spec.placement == nil", "spec.placement", fsvc.Spec.Placement)
		return map[string]v1beta1.ClusterPreferences{}, nil
	}

	var candidates []string
	if fsvc.Spec.Placement.Clusters != nil {
		lg.Info("placement.clusters found", "fsvc", fsvc.Spec.Placement.Clusters)
		for _, c := range fsvc.Spec.Placement.Clusters {
			candidates = append(candidates, c.Name)
		}
	} else {
		lg.Info("placement.clusterSelector found", "spec.placement.clusterSelector", fsvc.Spec.Placement.ClusterSelector)
		sel, err := metav1.LabelSelectorAsSelector(fsvc.Spec.Placement.ClusterSelector)
		if err != nil {
			lg.Error(err, "placement.clusterSelector")
			return nil, err
		}
		cl := &fedcorev1b1.KubeFedClusterList{}
		if err := r.List(ctx, cl, &client.ListOptions{
			Namespace:     wfc.Spec.KubeFedNamespace,
			LabelSelector: sel,
		}); err != nil {
			return nil, err
		}
		for _, c := range cl.Items {
			candidates = append(candidates, c.Name)
		}
	}

	var clusters []string
	cl := &fedcorev1b1.KubeFedClusterList{}
	if err := r.List(ctx, cl, &client.ListOptions{
		Namespace: wfc.Spec.KubeFedNamespace,
	}); err != nil {
		return nil, err
	}
	registered := map[string]struct{}{}
	for _, c := range cl.Items {
		registered[c.Name] = struct{}{}
	}
	dedup := map[string]int{}
	for _, c := range candidates {
		if _, ok := registered[c]; !ok {
			continue
		}
		dedup[c] += 1
		if dedup[c] == 1 {
			clusters = append(clusters, c)
		}
	}
	lg.Info("available clusters", "clusters", clusters)

	optimizeFn, ok := slpOptimizeFuncCollection[*wfc.Spec.LoadBalancing.Optimizer.Method]
	if !ok {
		return nil, fmt.Errorf("invalid method \"%v\"", wfc.Spec.LoadBalancing.Optimizer.Method)
	}
	cps, err := optimizeFn(ctx, clusters, wfc.Spec.LoadBalancing.Optimizer, fsvc)
	if err != nil {
		return nil, err
	}
	lg.Info("optimize weights", "weights", cps)
	return cps, nil
}

type slpOptimizeFunc func(ctx context.Context, clusters []string, settings *v1beta1.SLPOptimizerSettings, fsvc *structuredFederatedService) (map[string]v1beta1.ClusterPreferences, error)

var slpOptimizeFuncCollection = map[v1beta1.SLPOptimizerMethod]slpOptimizeFunc{
	v1beta1.SLPOptimizerMethodRoundRobin: slpOptimizeFnRoundRobin,
	// v1beta1.SLPOptimizerMethodWAO: slpOptimizeFnWAO, // TODO
}

func slpOptimizeFnRoundRobin(_ context.Context, clusters []string, _ *v1beta1.SLPOptimizerSettings, _ *structuredFederatedService) (map[string]v1beta1.ClusterPreferences, error) {
	cps := make(map[string]v1beta1.ClusterPreferences, len(clusters))
	for _, cl := range clusters {
		cps[cl] = v1beta1.ClusterPreferences{
			Weight: 1,
		}
	}
	return cps, nil
}
