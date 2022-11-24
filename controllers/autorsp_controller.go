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
	fedschedv1a1 "sigs.k8s.io/kubefed/pkg/apis/scheduling/v1alpha1"

	v1beta1 "github.com/Nedopro2022/waofed/api/v1beta1"
)

const (
	ControllerName = v1beta1.OperatorName + "-rspoptimizer-controller"
)

// RSPOptimizerReconciler reconciles a RSPOptimizer object
type RSPOptimizerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.kubefed.io,resources=kubefedclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=types.kubefed.io,resources=federateddeployments,verbs=get;list;watch
//+kubebuilder:rbac:groups=scheduling.kubefed.io,resources=replicaschedulingpreferences,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=waofed.bitmedia.co.jp,resources=waofedconfigs,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *RSPOptimizerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(newUnstructuredFederatedDeployment()).
		Owns(&fedschedv1a1.ReplicaSchedulingPreference{}).
		Complete(r)
}

// Reconcile moves the current state of the cluster closer to the desired state.
func (r *RSPOptimizerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx)
	lg.Info("Reconcile")

	// get WAOFedConfig
	wfc := &v1beta1.WAOFedConfig{}
	wfc.Name = v1beta1.WAOFedConfigName
	if err := r.Get(ctx, client.ObjectKeyFromObject(wfc), wfc); err != nil {
		lg.Error(err, fmt.Sprintf("unable to get WAOFedConfig %s", client.ObjectKeyFromObject(wfc)))
		return ctrl.Result{}, err
	}

	// get FederatedDeployment
	fdep := newUnstructuredFederatedDeployment()
	err := r.Get(ctx, req.NamespacedName, fdep)
	if errors.IsNotFound(err) {
		lg.Info("FederatedDeployment is already deleted")
		return ctrl.Result{}, nil
	}
	if err != nil {
		lg.Error(err, "unable to get FederatedDeployment")
		return ctrl.Result{}, err
	}
	fdeploy, err := convertToStructuredFederatedDeployment(fdep)
	if err != nil {
		lg.Error(err, "unable to convert FederatedDeployment")
	}

	// reconcile RSP
	if err := r.reconcileRSP(ctx, fdeploy, wfc); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RSPOptimizerReconciler) reconcileRSP(
	ctx context.Context, fdeploy *structuredFederatedDeployment, wfc *v1beta1.WAOFedConfig,
) error {
	lg := log.FromContext(ctx)
	lg.Info("reconcileRSP")

	skip := true
	if wfc.Spec.Scheduling.Selector.Any {
		// check selector.any
		skip = false
	} else if _, ok := fdeploy.GetAnnotations()[wfc.Spec.Scheduling.Selector.HasAnnotation]; ok {
		// check RSPOptimizer annotation exists in the FederatedDeployment
		// currently the value is ignored
		skip = false
	}

	if skip {
		// delete the associated RSP if no annotation in the FederatedDeployment
		//
		// An RSP associated with a FederatedDeployment and having an OwnerReference
		// will be deleted by GC when the FederatedDeployment is deleted.
		// However, in the case where the user deletes the RSPOptimizer annotation from
		// the FederatedDeployment causes problems.
		// Then the FederatedDeployment doesn't have any RSPOptimizer annotation,
		// so it should not be controlled by RSP, but RSP still exists and works.
		// Therefore, explicitly delete the RSP here.
		lg.Info("FederatedDeployment doesn't have RSPOptimizer annotation")
		// find RSP created by RSPOptimizer and delete it
		rsp := &fedschedv1a1.ReplicaSchedulingPreference{}
		rsp.SetNamespace(fdeploy.Namespace)
		rsp.SetName(fdeploy.Name)
		err := r.Get(ctx, client.ObjectKeyFromObject(rsp), rsp)
		if errors.IsNotFound(err) {
			lg.Info("RSP is already deleted")
			return nil
		}
		if err != nil {
			lg.Error(err, "unable to get RSP")
			return err
		}
		// check OwnerReference
		ctrlRef := metav1.GetControllerOf(rsp)
		compareRef := &metav1.OwnerReference{
			APIVersion: fdeploy.APIVersion,
			Kind:       fdeploy.Kind,
			Name:       fdeploy.Name,
		}
		if ctrlRef != nil && sameOwner(*ctrlRef, *compareRef) {
			// delete RSP
			lg.Info("RSP having a Controller OwnerReference implies the RSP was created by RSPOptimizer, so delete it")
			err := r.Delete(ctx, rsp)
			if errors.IsNotFound(err) {
				lg.Info("RSP is already deleted")
				return nil
			}
			if err != nil {
				lg.Error(err, "unable to delete RSP")
				return err
			}
		}
		return nil
	} else {
		// apply RSP if !skip
		rsp := &fedschedv1a1.ReplicaSchedulingPreference{}
		rsp.SetNamespace(fdeploy.Namespace)
		rsp.SetName(fdeploy.Name)
		op, err := ctrl.CreateOrUpdate(ctx, r.Client, rsp, func() error {
			// set labels
			rsp.Labels = map[string]string{
				"app.kubernetes.io/created-by": ControllerName,
			}
			// set RSP spec except clusters
			rsp.Spec = fedschedv1a1.ReplicaSchedulingPreferenceSpec{
				TargetKind:                   fdeploy.Kind,
				TotalReplicas:                *fdeploy.Spec.Template.Spec.Replicas,
				Rebalance:                    true,
				IntersectWithClusterSelector: false,
				Clusters:                     nil,
			}
			// set RSP clusters
			lg.Info("optimize cluster weights", "method", wfc.Spec.Scheduling.Optimizer.Method)
			clusters, err := r.optimizeClusterWeights(ctx, fdeploy, wfc)
			if err != nil {
				return err
			}
			rsp.Spec.Clusters = clusters
			// set OwnerReference
			//
			// HACK: ctrl.SetControllerReference requires both owner and controlled to have scheme registration,
			// but structuredFederatedDeployment has no scheme and is hard to implement,
			// so use setControllerReference instead.
			//
			// if err := ctrl.SetControllerReference(fdeploy, rsp, r.Scheme); err != nil {
			// 	return err
			// }
			if err := setControllerReference(fdeploy, rsp); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			lg.Error(err, "unable to create or update RSP")
		}
		lg.Info("RSP operated", "op", op)
	}

	return nil
}

func (r *RSPOptimizerReconciler) optimizeClusterWeights(
	ctx context.Context, fdeploy *structuredFederatedDeployment, wfc *v1beta1.WAOFedConfig,
) (map[string]fedschedv1a1.ClusterPreferences, error) {
	lg := log.FromContext(ctx)
	lg.Info("optimizeClusterWeights")

	// get cluster clusters
	//   if has placement.clusters field, the specified clusters should be candidates
	//   if only has placement.clusterSelector field, the selected clusters should be candidates
	//   if no placement field, no (or all) clusters should be candidates (consider making it configurable)
	if fdeploy.Spec.Placement == nil || (fdeploy.Spec.Placement.Clusters == nil && fdeploy.Spec.Placement.ClusterSelector == nil) {
		lg.Info("no scheduling as spec.placement == nil", "spec.placement", fdeploy.Spec.Placement)
		return map[string]fedschedv1a1.ClusterPreferences{}, nil
	}

	var candidates []string
	if fdeploy.Spec.Placement.Clusters != nil {
		// placement.clusters is specified
		lg.Info("placement.clusters found")
		for _, c := range fdeploy.Spec.Placement.Clusters {
			candidates = append(candidates, c.Name)
		}
	} else { // fdeploy.Spec.Placement.ClusterSelector != nil
		// placement.clusterSelector is specified
		lg.Info("placement.clusterSelector found")
		sel, err := metav1.LabelSelectorAsSelector(fdeploy.Spec.Placement.ClusterSelector)
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

	lg.Info("candidate clusters", "clusters", candidates)

	// optimize cluster weights
	optimizeFn, ok := optimizeFuncCollection[wfc.Spec.Scheduling.Optimizer.Method]
	if !ok {
		return nil, fmt.Errorf("invalid method \"%v\"", wfc.Spec.Scheduling.Optimizer.Method)
	}
	cps, err := optimizeFn(candidates)
	if err != nil {
		return nil, err
	}
	lg.Info("optimize weights", "weights", cps)
	return cps, nil
}

type optimizeFunc func(clusters []string) (map[string]fedschedv1a1.ClusterPreferences, error)

var optimizeFuncCollection = map[v1beta1.RSPOptimizerMethod]optimizeFunc{
	v1beta1.RSPOptimizerMethodRoundRobin: optimizeFnRoundRobin,
	v1beta1.RSPOptimizerMethodWAO:        optimizeFnWAO,
}

func optimizeFnRoundRobin(clusters []string) (map[string]fedschedv1a1.ClusterPreferences, error) {
	cps := make(map[string]fedschedv1a1.ClusterPreferences, len(clusters))
	for _, cl := range clusters {
		cps[cl] = fedschedv1a1.ClusterPreferences{
			MinReplicas: 0,
			MaxReplicas: nil,
			Weight:      1,
		}
	}
	return cps, nil
}

func optimizeFnWAO(clusters []string) (map[string]fedschedv1a1.ClusterPreferences, error) {
	cps := make(map[string]fedschedv1a1.ClusterPreferences, len(clusters))
	// TODO
	return cps, nil
}
