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
	fedcorecommon "sigs.k8s.io/kubefed/pkg/apis/core/common"
	fedcorev1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	fedschedv1a1 "sigs.k8s.io/kubefed/pkg/apis/scheduling/v1alpha1"

	v1beta1 "github.com/Nedopro2022/waofed/api/v1beta1"
)

const (
	ControllerName = v1beta1.OperatorName + "-autorsp-controller"
)

// AutoRSPReconciler reconciles a AutoRSP object
type AutoRSPReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.kubefed.io,resources=kubefedclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.kubefed.io,resources=kubefedclusters/status,verbs=get
//+kubebuilder:rbac:groups=types.kubefed.io,resources=federateddeployments,verbs=get;list;watch
//+kubebuilder:rbac:groups=scheduling.kubefed.io,resources=replicaschedulingpreferences,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=waofed.bitmedia.co.jp,resources=waofedconfigs,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *AutoRSPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(newUnstructuredFederatedDeployment()).
		Owns(&fedschedv1a1.ReplicaSchedulingPreference{}).
		Complete(r)
}

// Reconcile moves the current state of the cluster closer to the desired state.
func (r *AutoRSPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

func (r *AutoRSPReconciler) reconcileRSP(
	ctx context.Context, fdeploy *structuredFederatedDeployment, wfc *v1beta1.WAOFedConfig,
) error {
	lg := log.FromContext(ctx)
	lg.Info("reconcileRSP")

	// check AutoRSP annotation exists in the FederatedDeployment
	// currently the value is ignored
	_, ok := fdeploy.GetAnnotations()[*wfc.Spec.Scheduling.AutoRSPAnnotation]

	// delete the associated RSP if no annotation in the FederatedDeployment
	//
	// An RSP associated with a FederatedDeployment and having an OwnerReference
	// will be deleted by GC when the FederatedDeployment is deleted.
	// However, in the case where the user deletes the AutoRSP annotation from
	// the FederatedDeployment causes problems.
	// Then the FederatedDeployment doesn't have any AutoRSP annotation,
	// so it should not be controlled by RSP, but RSP still exists and works.
	// Therefore, explicitly delete the RSP here.
	if !ok {
		lg.Info("FederatedDeployment doesn't have AutoRSP annotation")
		// find RSP created by AutoRSP and delete it
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
			lg.Info("RSP having a Controller OwnerReference implies the RSP was created by AutoRSP, so delete it")
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
	}

	// apply RSP
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
			IntersectWithClusterSelector: true,
			Clusters:                     nil,
		}
		// set RSP clusters
		const AutoRSPMethod = "rr" // TODO
		lg.Info("optimize cluster weights", "method", AutoRSPMethod)
		optimizeFn, ok := optimizeFuncCollection[AutoRSPMethod]
		if !ok {
			return fmt.Errorf("invalid method %v", AutoRSPMethod)
		}
		clusters, err := r.optimizeClusterWeights(ctx, fdeploy, optimizeFn)
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

	return nil
}

func (r *AutoRSPReconciler) optimizeClusterWeights(
	ctx context.Context, fdeploy *structuredFederatedDeployment, optimizeFn optimizeFunc,
) (map[string]fedschedv1a1.ClusterPreferences, error) {
	lg := log.FromContext(ctx)
	lg.Info("optimizeClusterWeights")

	// get available clusters (status must be Ready)
	cls, err := r.listClusters(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("unable to get clusters: %w", err)
	}
	lg.Info("list clusters", "clusters", cls)
	// get optimal cluster weights
	cps, err := optimizeFn(cls)
	if err != nil {
		return nil, err
	}
	lg.Info("calc weights", "weights", cps)
	return cps, nil
}

func (r *AutoRSPReconciler) listClusters(ctx context.Context, statusReady bool) ([]string, error) {
	const kubefedNamespace = "kube-federation-system" // TODO
	clusters := &fedcorev1b1.KubeFedClusterList{}
	if err := r.List(ctx, clusters, &client.ListOptions{Namespace: kubefedNamespace}); err != nil {
		return nil, err
	}
	var a []string
	for _, cl := range clusters.Items {
		if !statusReady {
			// add the cluster regardless of status
			a = append(a, cl.Name)
			continue
		}
		if statusReady && cl.Status.Conditions[len(cl.Status.Conditions)-1].Type == fedcorecommon.ClusterReady {
			// add the cluster only if status is ready
			a = append(a, cl.Name)
			continue
		}
	}
	return a, nil
}

type optimizeFunc func(clusters []string) (map[string]fedschedv1a1.ClusterPreferences, error)

var optimizeFuncCollection = map[string]optimizeFunc{
	"rr":  optimizeFnRoundRobin,
	"wao": optimizeFnWAO,
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
