package controllers

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	krangv1alpha1 "github.com/dougbtv/krang/api/v1alpha1"
)

var _ = Describe("CNIPluginRegistration Controller", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		scheme     *runtime.Scheme
		k8sClient  client.Client
		reconciler *CNIPluginRegistrationReconciler
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		scheme = runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		Expect(krangv1alpha1.AddToScheme(scheme)).To(Succeed())

		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		reconciler = &CNIPluginRegistrationReconciler{
			Client: k8sClient,
			Scheme: scheme,
		}

		_ = os.Setenv("NODE_NAME", "test-node")
	})

	AfterEach(func() {
		cancel()
		_ = os.Unsetenv("NODE_NAME")
	})

	It("should create an install job if not present", func() {
		plugin := &krangv1alpha1.CNIPluginRegistration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tuning",
				Namespace: "default",
			},
			Spec: krangv1alpha1.CNIPluginRegistrationSpec{
				BinaryPath:     "/usr/src/bin/cni/tuning",
				CNINetworkType: "tuning",
				Image:          "busybox",
				ConfigJSON:     `{}`,
			},
		}
		Expect(k8sClient.Create(ctx, plugin)).To(Succeed())

		jobName := fmt.Sprintf("krang-install-%s-%s", plugin.Name, "test-node")
		jobKey := client.ObjectKey{Name: jobName, Namespace: plugin.Namespace}
		job := &batchv1.Job{}
		Expect(k8sClient.Get(ctx, jobKey, job)).To(HaveOccurred())

		req := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(plugin)}
		_, err := reconciler.Reconcile(ctx, req)
		Expect(err).NotTo(HaveOccurred())

		Expect(k8sClient.Get(ctx, jobKey, job)).To(Succeed())
		Expect(job.Name).To(Equal(jobName))
		Expect(job.Spec.Template.Spec.Containers[0].Image).To(Equal("busybox"))
	})
})
