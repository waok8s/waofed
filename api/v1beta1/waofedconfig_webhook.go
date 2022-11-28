package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
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

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-waofed-bitmedia-co-jp-v1beta1-waofedconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=waofed.bitmedia.co.jp,resources=waofedconfigs,verbs=create;update,versions=v1beta1,name=mwaofedconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &WAOFedConfig{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *WAOFedConfig) Default() {
	waofedconfiglog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-waofed-bitmedia-co-jp-v1beta1-waofedconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=waofed.bitmedia.co.jp,resources=waofedconfigs,verbs=create;update,versions=v1beta1,name=vwaofedconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &WAOFedConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *WAOFedConfig) ValidateCreate() error {
	waofedconfiglog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *WAOFedConfig) ValidateUpdate(old runtime.Object) error {
	waofedconfiglog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *WAOFedConfig) ValidateDelete() error {
	waofedconfiglog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
