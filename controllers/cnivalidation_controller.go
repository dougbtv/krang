package controllers

import (
	"context"
	"fmt"

	"github.com/dougbtv/krang/api/v1alpha1"
	"github.com/dougbtv/krang/pkg/logging"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CNIValidationReconciler reconciles a CNIValidation object
type CNIValidationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile handles the main validation logic
func (r *CNIValidationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logging.Verbosef("Reconciling CNIValidation: %s", req.NamespacedName)

	var validation v1alpha1.CNIValidation
	if err := r.Get(ctx, req.NamespacedName, &validation); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Log basic info for now, we'll add actual logic soon
	logging.Verbosef("Validating NetAttachDef: %s / Canary: %s",
		validation.Spec.NetworkRef.Name,
		validation.Spec.CanaryRef.Name,
	)

	// Placeholder status update
	validation.Status.Phase = "Validating"
	validation.Status.Message = fmt.Sprintf("Started validation for networkRef %s", validation.Spec.NetworkRef.Name)
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
