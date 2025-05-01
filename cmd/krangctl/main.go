package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	krangv1alpha1 "github.com/dougbtv/krang/api/v1alpha1"

	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	netdefclient "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
)

var manifestURLs = []string{
	"https://raw.githubusercontent.com/dougbtv/krang/main/manifests/crd/k8s.cni.cncf.io_cnipluginregistrations.yaml",
	"https://raw.githubusercontent.com/dougbtv/krang/main/manifests/crd/k8s.cni.cncf.io_cnimutationrequests.yaml",
	"https://raw.githubusercontent.com/dougbtv/krang/main/manifests/daemonset.yaml",
}

// var manifestURLs = []string{
// 	"https://gist.githubusercontent.com/dougbtv/b83cf0105e11085bb707dc4b08ac07d3/raw/a208c7daf620fd3b11d99dcb972162c3391f3acc/krang.yml",
// }

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
	rootCmd.AddCommand(newInstallCmd(&kubeconfig))
	rootCmd.AddCommand(newUninstallCmd(&kubeconfig))
	rootCmd.AddCommand(newUpgradeCmd(&kubeconfig))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newGetCmd(kubeconfig *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Retrieve krang-managed resources like plugins or configs",
	}
	cmd.AddCommand(newGetPluginsCmd(kubeconfig))
	cmd.AddCommand(newGetConfigsCmd(kubeconfig))
	return cmd
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

func newGetPluginsCmd(kubeconfig *string) *cobra.Command {
	var namespace string
	cmd := &cobra.Command{
		Use:   "plugins",
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

func newGetConfigsCmd(kubeconfig *string) *cobra.Command {
	var namespace string
	cmd := &cobra.Command{
		Use:   "configs",
		Short: "List NetworkAttachmentDefinitions (CNI configs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
			if err != nil {
				return fmt.Errorf("failed to load kubeconfig: %w", err)
			}

			netClient, err := netdefclient.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("failed to create net-attach-def client: %w", err)
			}

			var defs *netdefv1.NetworkAttachmentDefinitionList
			if namespace == "" {
				defs, err = netClient.K8sCniCncfIoV1().NetworkAttachmentDefinitions("").List(context.Background(), metav1.ListOptions{})
			} else {
				defs, err = netClient.K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).List(context.Background(), metav1.ListOptions{})
			}
			if err != nil {
				return fmt.Errorf("failed to list net-attach-defs: %w", err)
			}

			fmt.Printf("%-25s %-30s %-10s\n", "NAMESPACE", "NAME", "CNI TYPE")
			for _, def := range defs.Items {
				cniType := "-"
				if def.Spec.Config != "" {
					// naïvely parse CNI type from JSON config string
					var parsed map[string]interface{}
					_ = json.Unmarshal([]byte(def.Spec.Config), &parsed)
					if t, ok := parsed["type"].(string); ok {
						cniType = t
					}
				}
				fmt.Printf("%-25s %-30s %-10s\n", def.Namespace, def.Name, cniType)
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&namespace, "namespace", "", "Namespace to query (empty for all)")
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

func newInstallCmd(kubeconfig *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the krangd DaemonSet and CRDs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
			if err != nil {
				return fmt.Errorf("building kubeconfig: %w", err)
			}

			k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
			if err != nil {
				return fmt.Errorf("creating client: %w", err)
			}

			for _, url := range manifestURLs {
				fmt.Printf("📥 Applying manifest: %s\n", url)
				resp, err := http.Get(url)
				if err != nil {
					return fmt.Errorf("downloading manifest %s: %w", url, err)
				}
				defer resp.Body.Close()

				yamlDocs := utilyaml.NewYAMLOrJSONDecoder(resp.Body, 4096)
				for {
					var rawObj unstructured.Unstructured
					if err := yamlDocs.Decode(&rawObj); err != nil {
						if err == io.EOF {
							break
						}
						return fmt.Errorf("decoding YAML: %w", err)
					}

					if rawObj.Object == nil {
						continue
					}

					if err := k8sClient.Create(context.Background(), &rawObj); err != nil {
						fmt.Fprintf(os.Stderr, "warning: could not delete object: %v\n", err)
					}
				}
			}

			fmt.Println("⏳ Waiting for krangd daemonset to be ready...")
			return wait.PollImmediate(2*time.Second, 60*time.Second, func() (bool, error) {
				daemonset := &appsv1.DaemonSet{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: "krangd", Namespace: "kube-system"}, daemonset)
				if err != nil {
					return false, nil // keep polling
				}
				if daemonset.Status.NumberAvailable == daemonset.Status.DesiredNumberScheduled {
					fmt.Println("✅ krangd is ready!")
					return true, nil
				}
				return false, nil
			})
		},
	}
	return cmd
}

func newUninstallCmd(kubeconfig *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the krangd DaemonSet and CRDs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
			if err != nil {
				return fmt.Errorf("building kubeconfig: %w", err)
			}

			k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
			if err != nil {
				return fmt.Errorf("creating client: %w", err)
			}

			for _, url := range manifestURLs {
				fmt.Printf("🗑️ Deleting manifest: %s\n", url)
				resp, err := http.Get(url)
				if err != nil {
					return fmt.Errorf("downloading manifest %s: %w", url, err)
				}
				defer resp.Body.Close()

				yamlDocs := utilyaml.NewYAMLOrJSONDecoder(resp.Body, 4096)
				for {
					var rawObj unstructured.Unstructured
					if err := yamlDocs.Decode(&rawObj); err != nil {
						if err == io.EOF {
							break
						}
						return fmt.Errorf("decoding YAML: %w", err)
					}

					if rawObj.Object == nil {
						continue
					}

					if err := k8sClient.Delete(context.Background(), &rawObj); err != nil {
						fmt.Fprintf(os.Stderr, "warning: could not delete object: %v\n", err)
					}
				}
			}
			fmt.Println("🧹 krang uninstalled!")
			return nil
		},
	}
	return cmd
}

func newUpgradeCmd(kubeconfig *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade krang by uninstalling and reinstalling",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("🔄 Running upgrade...")
			if err := newUninstallCmd(kubeconfig).RunE(cmd, args); err != nil {
				return fmt.Errorf("uninstall failed: %w", err)
			}
			return newInstallCmd(kubeconfig).RunE(cmd, args)
		},
	}
	return cmd
}
