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
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	fedschedv1a1 "sigs.k8s.io/kubefed/pkg/apis/scheduling/v1alpha1"
	fedctrlutil "sigs.k8s.io/kubefed/pkg/controller/util"

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

//+kubebuilder:rbac:groups=types.kubefed.io,resources=federateddeployments,verbs=get;list;watch;
//+kubebuilder:rbac:groups=scheduling.kubefed.io,resources=replicaschedulingpreferences,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=waofed.bitmedia.co.jp,resources=waofedconfigs,verbs=get;list;watch;

// Reconcile moves the current state of the cluster closer to the desired state.
func (r *AutoRSPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx)
	lg.Info("Reconcile")

	// get WAOFedConfig
	wfc := &v1beta1.WAOFedConfig{}
	wfc.Name = v1beta1.WAOFedConfigName
	wfc.Namespace = v1beta1.WAOFedConfigNamespace
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
		delete := false
		for _, ref := range rsp.OwnerReferences {
			if ref.Controller != nil && *ref.Controller && ref.Name == fdeploy.Name {
				lg.Info("RSP having a Controller OwnerReference implies this RSP was created by AutoRSP, so delete it")
				delete = true
			}
		}
		if delete {
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
		lg.Info("RSP deleted")
		return nil
	}

	// apply RSP
	rsp := &fedschedv1a1.ReplicaSchedulingPreference{}
	rsp.SetNamespace(fdeploy.Namespace)
	rsp.SetName(fdeploy.Name)
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, rsp, func() error {
		rsp.Labels = map[string]string{
			"app.kubernetes.io/created-by": ControllerName,
		}
		spec := fedschedv1a1.ReplicaSchedulingPreferenceSpec{
			TargetKind:                   fdeploy.Kind,
			TotalReplicas:                *fdeploy.Spec.Template.Spec.Replicas,
			Rebalance:                    true,
			IntersectWithClusterSelector: false,
			Clusters:                     map[string]fedschedv1a1.ClusterPreferences{},
		}
		rsp.Spec = spec
		if err := setRSPClustersRoundRobin(ctx, rsp, fdeploy); err != nil {
			return err
		}
		// HACK: ctrl.SetControllerReference requires both owner and controlled to have scheme registration,
		// but structuredFederatedDeployment has no scheme and is hard to implement,
		// so set OwnerReferences manually instead.
		//
		// if err := ctrl.SetControllerReference(fdeploy, rsp, r.Scheme); err != nil {
		// 	return err
		// }
		rsp.OwnerReferences = append(rsp.OwnerReferences, metav1.OwnerReference{
			APIVersion:         fdeploy.APIVersion,
			Kind:               fdeploy.Kind,
			Name:               fdeploy.Name,
			UID:                fdeploy.UID,
			Controller:         pointer.Bool(true),
			BlockOwnerDeletion: pointer.BoolPtr(true),
		})
		return nil
	})
	if err != nil {
		lg.Error(err, "unable to create or update RSP")
	}
	lg.Info("RSP", "op", op)

	return nil
}

func setRSPClustersRoundRobin(
	ctx context.Context,
	rsp *fedschedv1a1.ReplicaSchedulingPreference,
	fdeploy *structuredFederatedDeployment,
) error {
	// TODO
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AutoRSPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(newUnstructuredFederatedDeployment()).
		// TODO: watch RSP to prevent user deletion
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
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              struct {
		Template  appsv1.Deployment                  `json:"template,omitempty"`
		Placement fedctrlutil.GenericPlacementFields `json:"placement,omitempty"`
	} `json:"spec,omitempty"`
}

func convertUnstructuredFieldToObject[T any](fieldName string, unstructuredObj map[string]any) (T, error) {
	var obj T
	v, ok := unstructuredObj[fieldName]
	if !ok {
		return obj, fmt.Errorf("could not get %s", fieldName)
	}

	// NOTE: Type assertion doesn't work, need to convert via JSON.
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

	// DEBUG
	// fmt.Printf("convertUnstructuredFieldToObject: %s\njson:\n%s\nobj:%#v\n", fieldName, p, obj)

	return obj, nil
}

func convertToStructuredFederatedDeployment(in *unstructured.Unstructured) (*structuredFederatedDeployment, error) {
	var out structuredFederatedDeployment

	if in.GroupVersionKind() != federatedDeploymentGVK {
		return nil, fmt.Errorf("wrong GVK: %v", in.GroupVersionKind())
	}
	out.TypeMeta = metav1.TypeMeta{
		Kind:       federatedDeploymentGVK.Kind,
		APIVersion: federatedDeploymentGVK.GroupVersion().Identifier(),
	}

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
