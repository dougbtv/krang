package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/dougbtv/krang/api/v1alpha1"
	"github.com/dougbtv/krang/pkg/logging"
	"k8s.io/client-go/util/retry"
)

// CNIPluginRegistrationReconciler reconciles a CNIPluginRegistration object
type CNIPluginRegistrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *CNIPluginRegistrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logging.Debugf("Reconciling CNIPluginRegistration: %s", req.NamespacedName)

	var reg v1alpha1.CNIPluginRegistration
	if err := r.Get(ctx, req.NamespacedName, &reg); err != nil {
		logging.Errorf("Unable to fetch CNIPluginRegistration: %v", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	localNodeName := os.Getenv("NODE_NAME")
	if localNodeName == "" {
		logging.Errorf("NODE_NAME environment variable not set")
		return ctrl.Result{}, fmt.Errorf("NODE_NAME not set")
	}

	now := metav1.Now()
	pluginName := reg.Name
	jobName := fmt.Sprintf("krang-install-%s-%s", pluginName, localNodeName)

	logging.Debugf("Checking for existing job: %s in namespace %s", jobName, req.Namespace)
	var job batchv1.Job
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: req.Namespace}, &job)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logging.Debugf("Job not found. Creating install job for plugin %s on node %s", pluginName, localNodeName)
			job := generateInstallJob(&reg, localNodeName, jobName, req.Namespace)
			if err := r.Create(ctx, job); err != nil {
				if apierrors.IsAlreadyExists(err) {
					logging.Debugf("Job already exists (race condition) for node %s", localNodeName)
					return ctrl.Result{}, nil
				}
				logging.Errorf("Failed to create job for node %s: %v", localNodeName, err)
				return ctrl.Result{}, err
			}
			logging.Verbosef("Created install job %s for node %s", jobName, localNodeName)

			nodeStatus := v1alpha1.NodePluginStatus{
				NodeName:  localNodeName,
				Ready:     false,
				Phase:     "installing",
				UpdatedAt: now,
			}

			err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				fresh := &v1alpha1.CNIPluginRegistration{}
				if err := r.Get(ctx, req.NamespacedName, fresh); err != nil {
					return err
				}

				logging.Debugf("Updating node status to installing for %s in CR %s/%s", localNodeName, req.Namespace, req.Name)

				found := false
				for i, n := range fresh.Status.Nodes {
					if n.NodeName == localNodeName {
						fresh.Status.Nodes[i] = nodeStatus
						found = true
						break
					}
				}
				if !found {
					fresh.Status.Nodes = append(fresh.Status.Nodes, nodeStatus)
				}

				return r.Status().Update(ctx, fresh)
			})

			if err != nil {
				logging.Errorf("Failed to update installing status for %s: %v", localNodeName, err)
				return ctrl.Result{}, err
			}

		} else {
			logging.Errorf("Failed to check job for node %s: %v", localNodeName, err)
			return ctrl.Result{}, err
		}
	} else {
		logging.Debugf("Job already exists for node %s", localNodeName)

		// Check if job completed successfully
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobComplete && cond.Status == v1.ConditionTrue {
				// Now check for binary existence
				pluginBinary := filepath.Base(reg.Spec.BinaryPath)
				pluginPath := filepath.Join("/opt/cni/bin", pluginBinary)
				_, statErr := os.Stat(pluginPath)
				ready := statErr == nil
				phase := "installing"
				if ready {
					phase = "ready"
					logging.Verbosef("Plugin binary %s found on disk for node %s", pluginPath, localNodeName)
				} else {
					logging.Debugf("Plugin binary %s not found yet on node %s", pluginPath, localNodeName)
				}

				nodeStatus := v1alpha1.NodePluginStatus{
					NodeName:  localNodeName,
					Ready:     ready,
					Phase:     phase,
					UpdatedAt: now,
				}

				err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
					fresh := &v1alpha1.CNIPluginRegistration{}
					if err := r.Get(ctx, req.NamespacedName, fresh); err != nil {
						return err
					}

					logging.Debugf("Updating node status for %s in CR %s/%s", localNodeName, req.Namespace, req.Name)

					found := false
					for i, n := range fresh.Status.Nodes {
						if n.NodeName == localNodeName {
							fresh.Status.Nodes[i] = nodeStatus
							found = true
							break
						}
					}
					if !found {
						fresh.Status.Nodes = append(fresh.Status.Nodes, nodeStatus)
					}

					return r.Status().Update(ctx, fresh)
				})

				if err != nil {
					logging.Errorf("Failed to update status for %s: %v", localNodeName, err)
					return ctrl.Result{}, err
				}

				logging.Verbosef("Successfully updated status for node %s", localNodeName)
				return ctrl.Result{}, nil
			}
		}

		logging.Debugf("Job not yet complete for node %s", localNodeName)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func generateInstallJob(reg *v1alpha1.CNIPluginRegistration, nodeName, jobName, namespace string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				"krang-install": reg.Name,
				"krang-node":    nodeName,
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: ptr(int32(60)),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"krang-install": reg.Name,
						"krang-node":    nodeName,
					},
				},
				Spec: v1.PodSpec{
					HostPID:     true,
					HostNetwork: true,
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": nodeName,
					},
					RestartPolicy: v1.RestartPolicyOnFailure,
					Containers: []v1.Container{
						{
							Name:    "installer",
							Image:   reg.Spec.Image,
							Command: []string{"cp", reg.Spec.BinaryPath, fmt.Sprintf("/host/opt/cni/bin/%s", reg.Name)},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "cnibin",
									MountPath: "/host/opt/cni/bin",
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: ptr(true),
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "cnibin",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/opt/cni/bin",
								},
							},
						},
					},
					Tolerations: []v1.Toleration{
						{
							Key:      "node-role.kubernetes.io/control-plane",
							Operator: v1.TolerationOpExists,
							Effect:   v1.TaintEffectNoSchedule,
						},
					},
				},
			},
		},
	}
}

func ptr[T any](v T) *T {
	return &v
}

func (r *CNIPluginRegistrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CNIPluginRegistration{}).
		Complete(r)
}
