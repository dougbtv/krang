package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	krangv1alpha1 "github.com/dougbtv/krang/api/v1alpha1"
)

func main() {
	var kubeconfig string
	if home := os.Getenv("HOME"); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", fmt.Sprintf("%s/.kube/config", home), "absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}

	rootCmd := &cobra.Command{Use: "krangctl"}
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", kubeconfig, "kubeconfig path")

	rootCmd.AddCommand(newCreateCmd(&kubeconfig))
	rootCmd.AddCommand(newGetCmd(&kubeconfig))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newCreateCmd(kubeconfig *string) *cobra.Command {
	var pluginName, namespace, image, cniType, binaryPath, config string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new CNIPluginRegistration",
		RunE: func(cmd *cobra.Command, args []string) error {
			k8sClient, err := newClient(*kubeconfig)
			if err != nil {
				return err
			}

			reg := &krangv1alpha1.CNIPluginRegistration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      pluginName,
				},
				Spec: krangv1alpha1.CNIPluginRegistrationSpec{
					Image:          image,
					CNINetworkType: cniType,
					BinaryPath:     binaryPath,
					ConfigJSON:     config,
				},
			}

			return k8sClient.Create(context.Background(), reg)
		},
	}

	cmd.Flags().StringVar(&pluginName, "name", "", "Name of the plugin (required)")
	cmd.Flags().StringVar(&namespace, "namespace", "kube-system", "Namespace for the plugin")
	cmd.Flags().StringVar(&image, "image", "", "Image for the plugin (required)")
	cmd.Flags().StringVar(&cniType, "cni-type", "", "CNI type name (required)")
	cmd.Flags().StringVar(&binaryPath, "binary-path", "", "Path to the plugin binary (required)")
	cmd.Flags().StringVar(&config, "config", "{}", "Raw CNI config JSON")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("image")
	cmd.MarkFlagRequired("cni-type")
	cmd.MarkFlagRequired("binary-path")

	return cmd
}

func newGetCmd(kubeconfig *string) *cobra.Command {
	var namespace string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "List all CNIPluginRegistrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			k8sClient, err := newClient(*kubeconfig)
			if err != nil {
				return err
			}

			var list krangv1alpha1.CNIPluginRegistrationList
			if err := k8sClient.List(context.Background(), &list, client.InNamespace(namespace)); err != nil {
				return err
			}

			for _, item := range list.Items {
				fmt.Printf("%s/%s\n", item.Namespace, item.Name)
				for _, node := range item.Status.Nodes {
					fmt.Printf("  - %s: %s (ready: %v)\n", node.NodeName, node.Phase, node.Ready)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "kube-system", "Namespace to query")
	return cmd
}

func newClient(kubeconfigPath string) (client.Client, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	_ = krangv1alpha1.AddToScheme(scheme)

	return client.New(cfg, client.Options{Scheme: scheme})
}
