package v1beta1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var waofedconfiglog = logf.Log.WithName("waofedconfig-resource")

func (r *WAOFedConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-waofed-bitmedia-co-jp-v1beta1-waofedconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=waofed.bitmedia.co.jp,resources=waofedconfigs,verbs=create;update,versions=v1beta1,name=mwaofedconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &WAOFedConfig{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *WAOFedConfig) Default() {
	waofedconfiglog.Info("default", "name", r.Name)
	if r.Spec.Scheduling != nil {
		r.defaultScheduling()
	}
	if r.Spec.LoadBalancing != nil {
		r.defaultLoadbalancing()
	}
}

func (r *WAOFedConfig) defaultScheduling() {
	waofedconfiglog.Info("default spec.scheduling", "name", r.Name)

	// selector
	if r.Spec.Scheduling.Selector == nil {
		r.Spec.Scheduling.Selector = &FederatedDeploymentSelector{}
	}
	if r.Spec.Scheduling.Selector.Any == nil {
		r.Spec.Scheduling.Selector.Any = pointer.Bool(false)
	}
	if r.Spec.Scheduling.Selector.HasAnnotation == nil {
		r.Spec.Scheduling.Selector.HasAnnotation = pointer.String(DefaultRSPOptimizerAnnotation)
	}

	// optimizer
	if r.Spec.Scheduling.Optimizer == nil {
		r.Spec.Scheduling.Optimizer = &RSPOptimizerSettings{}
	}
	if r.Spec.Scheduling.Optimizer.Method == nil {
		r.Spec.Scheduling.Optimizer.Method = (*RSPOptimizerMethod)(pointer.String(RSPOptimizerMethodRoundRobin))
	}

}

func (r *WAOFedConfig) defaultLoadbalancing() {
	waofedconfiglog.Info("default spec.loadbalancing", "name", r.Name)

	// TODO
}

//+kubebuilder:webhook:path=/validate-waofed-bitmedia-co-jp-v1beta1-waofedconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=waofed.bitmedia.co.jp,resources=waofedconfigs,verbs=create;update;delete,versions=v1beta1,name=vwaofedconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &WAOFedConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *WAOFedConfig) ValidateCreate() error {
	waofedconfiglog.Info("validate create", "name", r.Name)

	if err := r.validateResource(); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *WAOFedConfig) ValidateUpdate(old runtime.Object) error {
	waofedconfiglog.Info("validate update", "name", r.Name)

	if err := r.validateResource(); err != nil {
		return err
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *WAOFedConfig) ValidateDelete() error {
	waofedconfiglog.Info("validate delete", "name", r.Name)

	// TODO: check whether any FederatedDeployment is using WAOFed

	return nil
}

func (r *WAOFedConfig) validateResource() error {
	if err := r.validateName(); err != nil {
		return err
	}
	if err := r.validateKubeFedNS(); err != nil {
		return err
	}
	if r.Spec.Scheduling != nil {
		if err := r.validateScheduling(); err != nil {
			return err
		}
	}
	if r.Spec.LoadBalancing != nil {
		if err := r.validateLoadbalancing(); err != nil {
			return err
		}
	}
	return nil
}

func (r *WAOFedConfig) validateName() error {
	if r.Name != WAOFedConfigName {
		return fmt.Errorf("name must be %s", WAOFedConfigName)
	}
	return nil
}

func (r *WAOFedConfig) validateKubeFedNS() error {
	if r.Spec.KubeFedNamespace == "" {
		return fmt.Errorf("kubefedNamespace must be set")
	}
	return nil
}

func (r *WAOFedConfig) validateScheduling() error {
	// NOTE: the defaulting webhook ensures method != nil
	if _, ok := rspOptimizerMethodCollection[*r.Spec.Scheduling.Optimizer.Method]; !ok {
		return fmt.Errorf("invalid spec.scheduling.optimizer.method %s", *r.Spec.Scheduling.Optimizer.Method)
	}
	return nil
}

func (r *WAOFedConfig) validateLoadbalancing() error {
	// TODO
	return nil
}
