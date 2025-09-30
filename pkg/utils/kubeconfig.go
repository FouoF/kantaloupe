package utils

import (
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

func BuildLocalClusterConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		klog.V(2).InfoS("Using kubeconfig", "path", kubeconfigPath)
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			klog.ErrorS(err, "Failed to build config from kubeconfig")
			return nil, err
		}
		return config, nil
	}
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" && homedir.HomeDir() != "" {
		kubeconfig = homedir.HomeDir() + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			klog.ErrorS(err, "Failed to build config from kubeconfig")
			return nil, err
		}
	}
	return config, nil
}
