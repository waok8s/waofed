package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OperatorName = "waofed"

	DefaultRSPOptimizerAnnotation = "waofed.bitmedia.co.jp/scheduling"

	// WAOFedConfigName specifies the name of the only instance of WAOFedConfig that exists in the cluster.
	WAOFedConfigName = "default"

	waoEstimatorDefaultNamespace = "default"
	waoEstimatorDefaultName      = "default"
)

type RSPOptimizerMethod string

const (
	RSPOptimizerMethodRoundRobin = "rr"
	RSPOptimizerMethodWAO        = "wao"
)

type RSPOptimizerSettings struct {
	// Method specifies the method name to use. (default: "rr")
	// +optional
	Method *RSPOptimizerMethod `json:"method,omitempty"`

	// WAOEstimators specifies WAO-Estimator settings for member clusters.
	// Required when method "wao" is specified.
	//
	// e.g. { cluster1: {endpoint: "http://localhost:5657"}, cluster2: {endpoint: "http://localhost:5658"} }
	//
	// +optional
	WAOEstimators map[string]*WAOEstimatorSetting `json:"waoEstimators,omitempty"`
}

type WAOEstimatorSetting struct {
	// Endpoint specifies WAO-Estimator API endpoint.
	// e.g. "http://localhost:5657"
	Endpoint string `json:"endpoint"`
	// Namespace specifies Estimator resource namespace. (default: "default")
	Namespace string `json:"namespace,omitempty"`
	// Name specifies Estimator resource name. (default: "default")
	Name string `json:"name,omitempty"`
}

type FederatedDeploymentSelector struct {
	// Any matches any FederatedDeployment when set to true. (default: false)
	// +optional
	Any *bool `json:"any,omitempty"`
	// HasAnnotation specifies the annotation name within the FederatedDeployment to select. (default: "waofed.bitmedia.co.jp/scheduling")
	// +optional
	HasAnnotation *string `json:"hasAnnotation,omitempty"`
}

type SchedulingSettings struct {
	// Selector specifies the conditions that for FederatedDeployments to be affected by WAOFed.
	// +optional
	Selector *FederatedDeploymentSelector `json:"selector,omitempty"`
	// Optimizer owns optimizer settings that control how WAOFed generates ReplicaSchedulingPreferences.
	// +optional
	Optimizer *RSPOptimizerSettings `json:"optimizer,omitempty"`
}

type LoadBalancingSettings struct {
	// TODO
}

// WAOFedConfigSpec defines the desired state of WAOFedConfig
type WAOFedConfigSpec struct {
	// KubeFedNamespace specifies the KubeFed namespace used to check KubeFedCluster resources to get the list of clusters.
	KubeFedNamespace string `json:"kubefedNamespace,omitempty"`

	// Scheduling owns scheduling settings.
	// +optional
	Scheduling *SchedulingSettings `json:"scheduling,omitempty"`

	// LoadBalancing owns load balancing settings.
	// +optional
	LoadBalancing *LoadBalancingSettings `json:"loadbalancing,omitempty"`
}

// WAOFedConfigStatus defines the observed state of WAOFedConfig
type WAOFedConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster,shortName=waofed;wfc

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
