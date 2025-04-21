package controllers_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	krangv1alpha1 "github.com/dougbtv/krang/api/v1alpha1"
	"github.com/dougbtv/krang/controllers"
)

var _ = Describe("CNIMutationRequest Controller", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		scheme     *runtime.Scheme
		k8sClient  client.Client
		reconciler *controllers.CNIMutationRequestReconciler
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		scheme = runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		Expect(krangv1alpha1.AddToScheme(scheme)).To(Succeed())

		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		reconciler = &controllers.CNIMutationRequestReconciler{
			Client:        k8sClient,
			Scheme:        scheme,
			LocalNodeName: "test-node",
		}

		_ = os.MkdirAll("/tmp/test-cni-results", 0755)
		_ = os.Setenv("CNI_RESULTS_DIR", "/tmp/test-cni-results")
	})

	AfterEach(func() {
		cancel()
		_ = os.RemoveAll("/tmp/test-cni-results")
	})

	It("should skip pods not on this node", func() {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "demopod",
				Namespace: "default",
				Labels: map[string]string{
					"app": "demotuning",
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "other-node",
			},
		}
		Expect(k8sClient.Create(ctx, pod)).To(Succeed())

		mut := &krangv1alpha1.CNIMutationRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mutate-1",
				Namespace: "default",
			},
			Spec: krangv1alpha1.CNIMutationRequestSpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "demotuning"},
				},
				CNIConfig: `{ "cniVersion": "0.4.0", "name": "mutate", "plugins": [{"type": "noop"}]}`,
			},
		}
		Expect(k8sClient.Create(ctx, mut)).To(Succeed())

		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(mut)})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should look for CNI result file", func() {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mypod",
				Namespace: "default",
				Labels:    map[string]string{"app": "demotuning"},
			},
			Spec: corev1.PodSpec{
				NodeName: "test-node",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{
					ContainerID: "containerd://deadbeef",
				}},
			},
		}
		Expect(k8sClient.Create(ctx, pod)).To(Succeed())

		// Make a fake CNI result file
		fakeResult := map[string]string{
			"netns":  "/var/run/netns/fake",
			"ifName": "eth0",
		}
		content, _ := json.Marshal(fakeResult)
		_ = os.WriteFile(filepath.Join("/tmp/test-cni-results", "multus-cni-network-deadbeef-eth0"), content, 0644)

		mut := &krangv1alpha1.CNIMutationRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mutate-2",
				Namespace: "default",
			},
			Spec: krangv1alpha1.CNIMutationRequestSpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "demotuning"},
				},
				Interface: "eth0",
				CNIConfig: `{ "cniVersion": "0.4.0", "name": "mutate", "plugins": [{"type": "noop"}]}`,
			},
		}
		Expect(k8sClient.Create(ctx, mut)).To(Succeed())

		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(mut)})
		Expect(err).NotTo(HaveOccurred())
	})
})
