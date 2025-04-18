package controllers

import (
	"context"

	v1alpha1 "github.com/dougbtv/krang/api/v1alpha1"
	"github.com/dougbtv/krang/pkg/logging"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CNIPluginRegistrationReconciler reconciles a CNIPluginRegistration object
type CNIPluginRegistrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is the main logic
func (r *CNIPluginRegistrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logging.Debugf("Reconciling CNIPluginRegistration: %s", req.NamespacedName)

	var reg v1alpha1.CNIPluginRegistration
	if err := r.Get(ctx, req.NamespacedName, &reg); err != nil {
		logging.Errorf("Unable to fetch CNIPluginRegistration: %v", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logging.Verbosef("CNIPluginRegistration loaded: plugin=%s, image=%s", reg.Spec.CNINetworkType, reg.Spec.Image)

	// TODO: Implement node-local plugin sync (e.g., download image, copy binary)

	return ctrl.Result{}, nil
}

// SetupWithManager wires it to the manager
func (r *CNIPluginRegistrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CNIPluginRegistration{}).
		Complete(r)
}
