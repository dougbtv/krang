package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// CNIMutationRequestSpec defines the desired mutation behavior
type CNIMutationRequestSpec struct {
	PodSelector    metav1.LabelSelector `json:"podSelector"`
	CNINetworkType string               `json:"cniType"`   // e.g. "bpfman", "sysctl-manager"
	Interface      string               `json:"interface"` // Optional: which interface
	CNIConfig      string               `json:"config"`    // Raw CNI JSON config

	// Arbitrary plugin-specific arguments
	Args runtime.RawExtension `json:"args,omitempty"`
}

// CNIMutationRequestStatus reflects success/failure of execution
type CNIMutationRequestStatus struct {
	Phase      string             `json:"phase,omitempty"` // Pending, Processing, Complete, Failed
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type CNIMutationRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CNIMutationRequestSpec   `json:"spec,omitempty"`
	Status CNIMutationRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type CNIMutationRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CNIMutationRequest `json:"items"`
}
