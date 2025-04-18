package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	GroupVersion = schema.GroupVersion{
		Group:   "k8s.cni.cncf.io",
		Version: "v1alpha1",
	}

	SchemeBuilder = runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(GroupVersion,
			&CNIMutationRequest{},
			&CNIMutationRequestList{},
			&CNIPluginRegistration{},
			&CNIPluginRegistrationList{},
		)
		metav1.AddToGroupVersion(scheme, GroupVersion)
		return nil
	})

	AddToScheme = SchemeBuilder.AddToScheme
)
