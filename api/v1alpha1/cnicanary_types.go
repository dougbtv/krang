package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CNICanarySpec defines the desired configuration for a connectivity test
type CNICanarySpec struct {
	// What mode to use for deployment: "daemon" or "split-pair"
	// +kubebuilder:validation:Enum=daemon;split-pair
	Mode string `json:"mode"`

	// The network to test, as a reference to a NetAttachDef
	// Reference to a NetworkAttachmentDefinition to validate
	NetworkRef corev1.ObjectReference `json:"networkRef"`

	// Interface to use for test
	// +optional
	Interface string `json:"interface,omitempty"`

	// Connectivity rule (I'm not exactly sure what I'll put here yet, but for now)
	// +optional
	ConnectivityRule string `json:"connectivityRule,omitempty"`
}

// CNICanaryStatus defines the observed state of the canary test
type CNICanaryStatus struct {
	Phase       string      `json:"phase,omitempty"` // Pending, Running, Success, Failed
	Message     string      `json:"message,omitempty"`
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// Connectivity success from pods
	// +optional
	Connectivity map[string]bool `json:"connectivity,omitempty"`
}

// CNICanary is the Schema for CNI canary connectivity testing
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type CNICanary struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CNICanarySpec   `json:"spec,omitempty"`
	Status CNICanaryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type CNICanaryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CNICanary `json:"items"`
}
