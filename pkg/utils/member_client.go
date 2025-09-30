package utils

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clustercrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
	generatedclient "github.com/dynamia-ai/kantaloupe/api/crd/generated/clientset/versioned"
)

// ClusterClient stands for a cluster Clientset for the given member cluster.
type ClusterClient struct {
	KubeClient      *kubeclientset.Clientset
	GeneratedClient *generatedclient.Clientset
	ClusterName     string
}

// DynamicClusterClient stands for a dynamic client for the given member cluster.
type DynamicClusterClient struct {
	DynamicClientSet dynamic.Interface
	ClusterName      string
}

// Config holds the common attributes that can be passed to a Kubernetes client on
// initialization.

// ClientOption holds the attributes that should be injected to a Kubernetes client.
type ClientOption struct {
	// QPS indicates the maximum QPS to the master from this client.
	// If it's zero, the created RESTClient will use DefaultQPS: 5
	QPS float32

	// Burst indicates the maximum burst for throttle.
	// If it's zero, the created RESTClient will use DefaultBurst: 10.
	Burst int
}

func ClusterKubeconfig(clusterName string, client client.Client, clientOption *ClientOption) (*rest.Config, error) {
	kubeconfig, err := BuildClusterConfig(clusterName, clusterGetter(client), secretGetter(client))
	if err != nil {
		return nil, err
	}

	if kubeconfig != nil {
		if clientOption != nil {
			kubeconfig.QPS = clientOption.QPS
			kubeconfig.Burst = clientOption.Burst
		}
	}

	return kubeconfig, nil
}

// NewClusterClientSet returns a ClusterClient for the given member cluster.
func NewClusterClientSet(clusterName string, client client.Client, clientOption *ClientOption) (*ClusterClient, error) {
	kubeconfig, err := BuildClusterConfig(clusterName, clusterGetter(client), secretGetter(client))
	if err != nil {
		return nil, err
	}

	clusterClientSet := ClusterClient{ClusterName: clusterName}

	if kubeconfig != nil {
		if clientOption != nil {
			kubeconfig.QPS = clientOption.QPS
			kubeconfig.Burst = clientOption.Burst
		}
		clusterClientSet.KubeClient = kubeclientset.NewForConfigOrDie(kubeconfig)
		clusterClientSet.GeneratedClient = generatedclient.NewForConfigOrDie(kubeconfig)
	}
	return &clusterClientSet, nil
}

// NewClusterDynamicClientSet returns a dynamic client for the given member cluster.
func NewClusterDynamicClientSet(clusterName string, client client.Client) (*DynamicClusterClient, error) {
	clusterConfig, err := BuildClusterConfig(clusterName, clusterGetter(client), secretGetter(client))
	if err != nil {
		return nil, err
	}
	clusterClientSet := DynamicClusterClient{ClusterName: clusterName}

	if clusterConfig != nil {
		clusterClientSet.DynamicClientSet = dynamic.NewForConfigOrDie(clusterConfig)
	}
	return &clusterClientSet, nil
}

// BuildClusterConfig return rest config for member cluster.
func BuildClusterConfig(clusterName string,
	clusterGetter func(string) (*clustercrdv1alpha1.Cluster, error),
	secretGetter func(string, string) (*corev1.Secret, error),
) (*rest.Config, error) {
	cluster, err := clusterGetter(clusterName)
	if err != nil {
		return nil, err
	}

	if cluster.Spec.SecretRef == nil {
		return nil, fmt.Errorf("cluster %s does not have a secret", clusterName)
	}

	secret, err := secretGetter(cluster.Spec.SecretRef.Namespace, cluster.Spec.SecretRef.Name)
	if err != nil {
		return nil, err
	}

	configBytes, ok := secret.Data["config"]
	if !ok {
		return nil, errors.New("the secret data is empty")
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(configBytes)
	if err != nil {
		return nil, err
	}

	return clientConfig.ClientConfig()
}

func clusterGetter(client client.Client) func(string) (*clustercrdv1alpha1.Cluster, error) {
	return func(clusterName string) (*clustercrdv1alpha1.Cluster, error) {
		cluster := &clustercrdv1alpha1.Cluster{}
		if err := client.Get(context.TODO(), types.NamespacedName{Name: clusterName}, cluster); err != nil {
			return nil, err
		}
		return cluster, nil
	}
}

func secretGetter(client client.Client) func(string, string) (*corev1.Secret, error) {
	return func(namespace, name string) (*corev1.Secret, error) {
		secret := &corev1.Secret{}
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, secret)
		return secret, err
	}
}
