package controllers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/libcni"
	krangv1alpha1 "github.com/dougbtv/krang/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CNIMutationRequestReconciler reconciles a CNIMutationRequest object
type CNIMutationRequestReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	LocalNodeName string
}

func (r *CNIMutationRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling CNIMutationRequest", "name", req.NamespacedName)

	var mutateReq krangv1alpha1.CNIMutationRequest
	if err := r.Get(ctx, req.NamespacedName, &mutateReq); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Find matching pods
	var podList corev1.PodList
	selector, _ := metav1.LabelSelectorAsSelector(&mutateReq.Spec.PodSelector)
	if err := r.List(ctx, &podList, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return ctrl.Result{}, err
	}

	for _, pod := range podList.Items {
		if pod.Spec.NodeName != r.LocalNodeName {
			continue
		}
		if len(pod.Status.ContainerStatuses) == 0 {
			continue
		}
		containerID := strings.TrimPrefix(pod.Status.ContainerStatuses[0].ContainerID, "containerd://")

		// Search for the matching results file
		entries, err := os.ReadDir("/var/lib/cni/results")
		if err != nil {
			logger.Error(err, "Unable to list CNI results directory")
			continue
		}

		var resultFile string
		for _, entry := range entries {
			if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), "-eth0") {
				continue
			}
			path := filepath.Join("/var/lib/cni/results", entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if strings.Contains(string(data), pod.Name) && strings.Contains(string(data), pod.Namespace) {
				resultFile = path
				break
			}
		}

		if resultFile == "" {
			logger.Info("No matching result file found", "pod", pod.Name)
			continue
		}

		raw, err := os.ReadFile(resultFile)
		if err != nil {
			logger.Error(err, "Could not read CNI result cache file", "path", resultFile)
			continue
		}

		var cached struct {
			NetNS  string `json:"netns"`
			IfName string `json:"ifName"`
		}
		if err := json.Unmarshal(raw, &cached); err != nil {
			logger.Error(err, "Failed to parse CNI result cache")
			continue
		}

		netnsPath := cached.NetNS
		ifName := cached.IfName
		if mutateReq.Spec.Interface != "" {
			ifName = mutateReq.Spec.Interface
		}

		rt := &libcni.RuntimeConf{
			ContainerID: containerID,
			NetNS:       netnsPath,
			IfName:      ifName,
			Args:        [][2]string{},
		}

		confList, err := libcni.ConfListFromBytes([]byte(mutateReq.Spec.CNIConfig))
		if err != nil {
			logger.Error(err, "Failed to parse CNI config")
			continue
		}

		cniPaths := []string{"/opt/cni/bin"}
		cni := libcni.NewCNIConfigWithCacheDir(cniPaths, "/etc/cni/net.d", nil)

		result, err := cni.AddNetworkList(context.Background(), confList, rt)
		if err != nil {
			logger.Error(err, "CNI Add failed")
			continue
		}

		logger.Info("CNI ADD completed", "pod", pod.Name, "result", result)
	}

	return ctrl.Result{}, nil
}

func (r *CNIMutationRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krangv1alpha1.CNIMutationRequest{}).
		Complete(r)
}
