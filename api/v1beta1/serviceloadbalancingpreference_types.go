package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceLoadbalancingPreferenceSpec defines the desired state of ServiceLoadbalancingPreference
type ServiceLoadbalancingPreferenceSpec struct {
	// Clusters maps between cluster names and preference weight settings in these clusters.
	// "*" (if provided) applies to all clusters if an explicit mapping is not provided.
	// Clusters without preferences should not have any access.
	Clusters map[string]ClusterPreferences `json:"clusters"`
}

// ClusterPreferences represent the weight of the service in a cluster.
type ClusterPreferences struct {
	// Weight is the weight in 64 bit **signed** integer.
	// Loadbalancer controllers using SLP should normalize the value.
	Weight int64 `json:"weight"`
}

// ServiceLoadbalancingPreferenceStatus defines the observed state of ServiceLoadbalancingPreference
type ServiceLoadbalancingPreferenceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=slp

// ServiceLoadbalancingPreference is the Schema for the serviceloadbalancingpreferences API
type ServiceLoadbalancingPreference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceLoadbalancingPreferenceSpec   `json:"spec,omitempty"`
	Status ServiceLoadbalancingPreferenceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceLoadbalancingPreferenceList contains a list of ServiceLoadbalancingPreference
type ServiceLoadbalancingPreferenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceLoadbalancingPreference `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceLoadbalancingPreference{}, &ServiceLoadbalancingPreferenceList{})
}
