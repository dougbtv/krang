package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

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

	rootCmd.AddCommand(newRegisterCmd(&kubeconfig))
	rootCmd.AddCommand(newUnregisterCmd(&kubeconfig))
	rootCmd.AddCommand(newGetCmd(&kubeconfig))
	rootCmd.AddCommand(newMutateCmd(&kubeconfig))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newRegisterCmd(kubeconfig *string) *cobra.Command {
	var pluginName, namespace, image, cniType, binaryPath, config string
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new CNIPluginRegistration",
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

func newUnregisterCmd(kubeconfig *string) *cobra.Command {
	var pluginName, namespace string
	cmd := &cobra.Command{
		Use:   "unregister",
		Short: "Unregister (delete) a CNIPluginRegistration",
		RunE: func(cmd *cobra.Command, args []string) error {
			k8sClient, err := newClient(*kubeconfig)
			if err != nil {
				return err
			}

			key := client.ObjectKey{
				Namespace: namespace,
				Name:      pluginName,
			}
			obj := &krangv1alpha1.CNIPluginRegistration{}
			if err := k8sClient.Get(context.Background(), key, obj); err != nil {
				return fmt.Errorf("failed to get registration: %w", err)
			}

			return k8sClient.Delete(context.Background(), obj)
		},
	}

	cmd.Flags().StringVar(&pluginName, "name", "", "Name of the plugin (required)")
	cmd.Flags().StringVar(&namespace, "namespace", "kube-system", "Namespace of the plugin")
	cmd.MarkFlagRequired("name")

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

			fmt.Printf("%-20s %-20s %-25s %-10s %-8s\n", "NAMESPACE", "NAME", "NODE", "PHASE", "READY")
			for _, item := range list.Items {
				for i, node := range item.Status.Nodes {
					// Print name only on first node line
					ns := item.Namespace
					name := item.Name
					if i > 0 {
						ns, name = "", ""
					}
					// Get the first 15 characters of the node name
					fmt.Printf("%-20s %-20s %-25s %-10s %-8v\n",
						ns, name, node.NodeName, node.Phase, node.Ready)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "kube-system", "Namespace to query")
	return cmd
}

func newMutateCmd(kubeconfig *string) *cobra.Command {
	var namespace, cniType, ifName, configPathOrContent, matchLabelsRaw string

	cmd := &cobra.Command{
		Use:   "mutate",
		Short: "Create a CNIMutationRequest to mutate a running pod",
		RunE: func(cmd *cobra.Command, args []string) error {
			k8sClient, err := newClient(*kubeconfig)
			if err != nil {
				return err
			}

			configData := configPathOrContent
			if _, err := os.Stat(configPathOrContent); err == nil {
				data, err := os.ReadFile(configPathOrContent)
				if err != nil {
					return fmt.Errorf("failed to read config file: %w", err)
				}
				configData = string(data)
			}

			matchLabels := map[string]string{}
			if matchLabelsRaw != "" {
				pairs := strings.Split(matchLabelsRaw, ",")
				for _, pair := range pairs {
					parts := strings.SplitN(pair, "=", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid matchlabel format: %q (expected key=value)", pair)
					}
					matchLabels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}

			mut := &krangv1alpha1.CNIMutationRequest{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: fmt.Sprintf("mutate-%s-", cniType),
					Namespace:    namespace,
				},
				Spec: krangv1alpha1.CNIMutationRequestSpec{
					CNINetworkType: cniType,
					Interface:      ifName,
					CNIConfig:      configData,
					PodSelector: metav1.LabelSelector{
						MatchLabels: matchLabels,
					},
				},
			}

			if err := k8sClient.Create(context.Background(), mut); err != nil {
				return fmt.Errorf("failed to create CNIMutationRequest: %w", err)
			}

			fmt.Printf("✅ CNIMutationRequest %q created in namespace %q\n", mut.Name, namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "kube-system", "Namespace to create the CNIMutationRequest")
	cmd.Flags().StringVar(&cniType, "cni-type", "", "CNI type for the mutation (required)")
	cmd.Flags().StringVar(&ifName, "interface", "eth0", "Target interface to mutate")
	cmd.Flags().StringVar(&configPathOrContent, "config", "", "Path to CNI config or inline JSON (required)")
	cmd.Flags().StringVar(&matchLabelsRaw, "matchlabels", "", "Comma-separated key=value pod label selector (required)")

	cmd.MarkFlagRequired("cni-type")
	cmd.MarkFlagRequired("config")
	cmd.MarkFlagRequired("matchlabels")

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
