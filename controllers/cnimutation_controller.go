package controllers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/libcni"
	krangv1alpha1 "github.com/dougbtv/krang/api/v1alpha1"
	"github.com/dougbtv/krang/pkg/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CNIMutationRequestReconciler reconciles a CNIMutationRequest object
type CNIMutationRequestReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	LocalNodeName string
}

func (r *CNIMutationRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logging.Verbosef("Reconciling CNIMutationRequest: %s", req.NamespacedName)

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
			logging.Errorf("Unable to list CNI results directory: %v", err)
			continue
		}

		// TODO: This whole bit about reading CNI cache results could be an entirely library.
		// Or it needs another approach, but for now, it has everything I need to say "this is how I exec a CNI plugin against a running netns"
		// Additionally, this is probably a slow way to do it. It's PoC style here.
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
			logging.Verbosef("No matching CNI result file found for pod %s", pod.Name)
			continue
		}

		raw, err := os.ReadFile(resultFile)
		if err != nil {
			logging.Errorf("Failed to read CNI result file: %v", err)
			continue
		}

		var cached struct {
			NetNS  string `json:"netns"`
			IfName string `json:"ifName"`
		}
		if err := json.Unmarshal(raw, &cached); err != nil {
			logging.Errorf("Failed to unmarshal CNI result file: %v", err)
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
			logging.Errorf("Failed to parse CNI config: %v", err)
			continue
		}

		cniPaths := []string{"/opt/cni/bin"}
		cni := libcni.NewCNIConfigWithCacheDir(cniPaths, "/etc/cni/net.d", nil)

		result, err := cni.AddNetworkList(context.Background(), confList, rt)
		if err != nil {
			logging.Errorf("CNI Add failed: %v", err)
			continue
		}

		logging.Verbosef("CNI ADD completed: pod: %s / result: %v", pod.Name, result)
	}

	return ctrl.Result{}, nil
}

func (r *CNIMutationRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krangv1alpha1.CNIMutationRequest{}).
		Complete(r)
}
