package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	fedschedv1a1 "sigs.k8s.io/kubefed/pkg/apis/scheduling/v1alpha1"
	fedctrlutil "sigs.k8s.io/kubefed/pkg/controller/util"

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
	_, ok := fdep.GetAnnotations()[AnnotAutoRSP]
	if !ok {
		lg.Info("FederatedDeployment doesn't have AutoRSP annotation")
		return ctrl.Result{}, nil
	}

	// // debug fdep
	// j, err := json.Marshal(fdep.Object)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("fdeploy: %s", j)

	fdeploy, err := convertToStructuredFederatedDeployment(fdep)
	if err != nil {
		lg.Error(err, "unable to convert FederatedDeployment")
	}
	if err := r.reconcileRSP(ctx, fdeploy); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AutoRSPReconciler) reconcileRSP(ctx context.Context, fdeploy *structuredFederatedDeployment) error {
	lg := log.FromContext(ctx)
	lg.Info("reconcileRSP")

	var _ fedschedv1a1.ReplicaSchedulingPreference

	// TODO

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AutoRSPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(newUnstructuredFederatedDeployment()).
		Complete(r)
}

var federatedDeploymentGVK = schema.GroupVersionKind{
	Group:   "types.kubefed.io",
	Kind:    "FederatedDeployment",
	Version: "v1beta1",
}

func newUnstructuredFederatedDeployment() *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(federatedDeploymentGVK)
	return u
}

type structuredFederatedDeployment struct {
	schema.GroupVersionKind
	metav1.ObjectMeta
	Spec struct {
		Template  appsv1.Deployment
		Placement fedctrlutil.GenericPlacementFields
	}
}

func convertUnstructuredFieldToObject[T any](fieldName string, unstructuredObj map[string]any) (T, error) {
	var obj T
	v, ok := unstructuredObj[fieldName]
	if !ok {
		return obj, fmt.Errorf("could not get %s", fieldName)
	}

	// Type assertion doesn't work, need to convert via JSON.
	//
	// obj, ok = v.(T)
	// if !ok { // always false
	// 	return obj, fmt.Errorf("bad type assertion")
	// }

	p, err := json.Marshal(&v)
	if err != nil {
		return obj, fmt.Errorf("could not encode %s: %v", fieldName, err)
	}
	if err := json.Unmarshal(p, &obj); err != nil {
		return obj, fmt.Errorf("could not decode %s: %v", fieldName, err)
	}

	// // debug
	// fmt.Printf("convertUnstructuredFieldToObject: %s\njson:\n%s\nobj:%#v\n", fieldName, p, obj)

	return obj, nil
}

func convertToStructuredFederatedDeployment(in *unstructured.Unstructured) (*structuredFederatedDeployment, error) {
	var out structuredFederatedDeployment

	if in.GroupVersionKind() != federatedDeploymentGVK {
		return nil, fmt.Errorf("wrong GVK: %v", in.GroupVersionKind())
	}
	out.GroupVersionKind = federatedDeploymentGVK

	objMeta, err := convertUnstructuredFieldToObject[*metav1.ObjectMeta]("metadata", in.Object)
	if err != nil {
		return nil, err
	}
	out.ObjectMeta = *objMeta

	v, ok := in.Object["spec"]
	if !ok {
		return nil, fmt.Errorf("could not get %s", "spec")
	}
	spec, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("could not encode %s", "spec")
	}

	objPlacement, err := convertUnstructuredFieldToObject[*fedctrlutil.GenericPlacementFields]("placement", spec)
	if err != nil {
		return nil, err
	}
	out.Spec.Placement = *objPlacement

	objDeployment, err := convertUnstructuredFieldToObject[*appsv1.Deployment]("template", spec)
	if err != nil {
		return nil, err
	}
	out.Spec.Template = *objDeployment

	return &out, nil
}
