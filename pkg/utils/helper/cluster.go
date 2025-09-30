package helper

import (
	"fmt"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	clustercrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
)

// IsClusterReady tells whether the cluster status in 'Ready' condition.
func IsClusterReady(clusterStatus *clustercrdv1alpha1.ClusterStatus) bool {
	for _, condition := range clusterStatus.Conditions {
		if condition.Type == clustercrdv1alpha1.ClusterConditionReady && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func MakeConfigSkipTLS(config *rest.Config) {
	if config == nil {
		return
	}
	config.CAData = nil
	config.Insecure = true
}

// GetCurrentNS fetch namespace the current pod running in. reference to client-go (config *inClusterClientConfig) Namespace() (string, bool, error).
func GetCurrentNS() (string, error) {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns, nil
	}

	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns, nil
		}
	}
	return "", fmt.Errorf("can not get namespace where pods running in")
}

func GetCurrentNSOrDefault() string {
	ns, err := GetCurrentNS()
	if err != nil {
		return "kantaloupe-system"
	}
	return ns
}
