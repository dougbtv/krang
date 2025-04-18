package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CNIPluginRegistrationSpec describes the plugin
type CNIPluginRegistrationSpec struct {
	CNINetworkType string `json:"cniType"`    // e.g. "bpfman"
	ConfigJSON     string `json:"config"`     // Raw CNI JSON config
	Image          string `json:"image"`      // e.g. ghcr.io/foo/sysctl-manager
	BinaryPath     string `json:"binaryPath"` // e.g. /plugins/sysctl-manager
}

type NodePluginStatus struct {
	NodeName  string      `json:"node"`
	Ready     bool        `json:"ready"`
	Message   string      `json:"message,omitempty"`
	UpdatedAt metav1.Time `json:"updatedAt"`
	Phase     string      `json:"phase,omitempty"` // installing, ready, failed
}

// CNIPluginRegistrationStatus shows plugin rollout state
type CNIPluginRegistrationStatus struct {
	Nodes []NodePluginStatus `json:"nodes"`
}

// +kubebuilder:object:root=true
type CNIPluginRegistration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CNIPluginRegistrationSpec   `json:"spec,omitempty"`
	Status CNIPluginRegistrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type CNIPluginRegistrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CNIPluginRegistration `json:"items"`
}
