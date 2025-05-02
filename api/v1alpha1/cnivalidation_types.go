package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CNIValidationSpec defines the desired validation input
// CNIValidationSpec describes the inputs to validate a CNI config
type CNIValidationSpec struct {
	// Reference to a NetworkAttachmentDefinition to validate
	NetworkRef corev1.ObjectReference `json:"networkRef"`

	// Reference to a CNICanary to validate
	// +optional
	CanaryRef corev1.ObjectReference `json:"canaryRef"`

	// Optional override for config instead of resolving from NetAttachDef
	// +optional
	ConfigOverride string `json:"config,omitempty"`
}

// CNIValidationStatus describes the results of the validation
type CNIValidationStatus struct {
	// Whether the referenced plugins were found on disk
	// +optional
	PluginsInstalled bool `json:"pluginsInstalled,omitempty"`

	// Whether the config was syntactically valid
	// +optional
	ConfigValid bool `json:"configValid,omitempty"`

	// Whether connectivity was tested
	// +optional
	ConfigTested bool `json:"configTested,omitempty"`

	// Any status message or failure explanation
	// +optional
	Message string `json:"message,omitempty"`

	// Plugin types parsed from config
	// +optional
	PluginTypes []string `json:"pluginTypes,omitempty"`

	// Validation timestamp
	// +optional
	ValidatedAt metav1.Time `json:"validatedAt,omitempty"`

	// Phase of validation lifecycle: Pending, Validating, Complete, Failed
	// +optional
	Phase string `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CNIValidation is the Schema for validating a NetworkAttachmentDefinition
type CNIValidation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CNIValidationSpec   `json:"spec,omitempty"`
	Status CNIValidationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CNIValidationList contains a list of CNIValidation
type CNIValidationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CNIValidation `json:"items"`
}
