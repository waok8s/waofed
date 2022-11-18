package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OperatorName = "waofed"
)

// WAOFedConfigSpec defines the desired state of WAOFedConfig
type WAOFedConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// TODO

	// Foo is an example field of WAOFedConfig. Edit waofedconfig_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// WAOFedConfigStatus defines the observed state of WAOFedConfig
type WAOFedConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// WAOFedConfig is the Schema for the waofedconfigs API
type WAOFedConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WAOFedConfigSpec   `json:"spec,omitempty"`
	Status WAOFedConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WAOFedConfigList contains a list of WAOFedConfig
type WAOFedConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WAOFedConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WAOFedConfig{}, &WAOFedConfigList{})
}
