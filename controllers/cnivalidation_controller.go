package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dougbtv/krang/api/v1alpha1"
	"github.com/dougbtv/krang/pkg/logging"
	netdefclient "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CNIValidationReconciler reconciles a CNIValidation object
type CNIValidationReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	NetDefClient netdefclient.Interface
}

// Reconcile handles the main validation logic
func (r *CNIValidationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logging.Verbosef("Reconciling CNIValidation: %s", req.NamespacedName)

	var validation v1alpha1.CNIValidation
	if err := r.Get(ctx, req.NamespacedName, &validation); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch NetAttachDef using netdefclient
	logging.Verbosef("validation network ref namespace and name: %s/%s", validation.Spec.NetworkRef.Namespace, validation.Spec.NetworkRef.Name)
	nad, err := r.NetDefClient.K8sCniCncfIoV1().NetworkAttachmentDefinitions(validation.Spec.NetworkRef.Namespace).Get(ctx, validation.Spec.NetworkRef.Name, metav1.GetOptions{})
	if err != nil {
		validation.Status.Phase = "Failed"
		validation.Status.Message = fmt.Sprintf("Failed to fetch NetworkAttachmentDefinition: %v", err)
		_ = r.Status().Update(ctx, &validation)
		return ctrl.Result{}, nil
	}

	typesFound := []string{}
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(nad.Spec.Config), &config); err != nil {
		validation.Status.Phase = "Failed"
		validation.Status.Message = fmt.Sprintf("Failed to parse config: %v", err)
		_ = r.Status().Update(ctx, &validation)
		return ctrl.Result{}, nil
	}

	var extractTypes func(interface{})
	extractTypes = func(data interface{}) {
		switch val := data.(type) {
		case map[string]interface{}:
			if t, ok := val["type"].(string); ok {
				typesFound = append(typesFound, t)
			}
			for _, v := range val {
				extractTypes(v)
			}
		case []interface{}:
			for _, item := range val {
				extractTypes(item)
			}
		}
	}
	extractTypes(config)

	// Check plugin registrations
	pluginsReady := true
	for _, plugin := range typesFound {
		var reg v1alpha1.CNIPluginRegistration
		key := client.ObjectKey{Name: plugin, Namespace: "kube-system"}
		if err := r.Get(ctx, key, &reg); err != nil {
			pluginsReady = false
			break
		}

		for _, node := range reg.Status.Nodes {
			if !node.Ready {
				pluginsReady = false
				break
			}
		}
	}

	logging.Verbosef("!validate Plugins installed: %v", pluginsReady)

	validation.Status.PluginTypes = typesFound
	validation.Status.ConfigValid = true
	validation.Status.PluginsInstalled = pluginsReady
	validation.Status.Phase = "Complete"
	validation.Status.Message = "Validation completed"

	if err := r.Status().Update(ctx, &validation); err != nil {
		logging.Errorf("Failed to update CNIValidation status: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CNIValidationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CNIValidation{}).
		Complete(r)
}
