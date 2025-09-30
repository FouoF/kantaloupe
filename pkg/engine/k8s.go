package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kantaloupeclusterv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
	kantaloupev1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/generated/clientset/versioned"
	kantaloupetypedclientset "github.com/dynamia-ai/kantaloupe/api/crd/generated/clientset/versioned/typed/cluster/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/gclient"
)

var (
	clientManager *ClientManager
	once          sync.Once
)

type Options struct {
	QPS        float32
	Burst      int
	Kubeconfig string
	TTL        time.Duration
}

const LocalCluster = "local-cluster"

func WithQPS(qps float32) func(*Options) {
	return func(o *Options) {
		o.QPS = qps
	}
}

func WithBurst(burst int) func(*Options) {
	return func(o *Options) {
		o.Burst = burst
	}
}

func WithKubeconfig(kubeconfig string) func(*Options) {
	return func(o *Options) {
		o.Kubeconfig = kubeconfig
	}
}

func WithTTL(ttl time.Duration) func(*Options) {
	return func(o *Options) {
		o.TTL = ttl
	}
}

type Client struct {
	Dynamic dynamic.Interface
	clientset.Interface
	client.Client
	kantaloupetypedclientset.ClusterV1alpha1Interface
}

type ClientManagerInterface interface {
	GeteClient(clusterName string) (*Client, error)
}

func (cli *Client) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return cli.Dynamic.Resource(resource)
}

func NewClientManager(opts ...func(*Options)) ClientManagerInterface {
	once.Do(func() {
		clientManager = newClientManager(opts...)
	})
	return clientManager
}

func newClientManager(opts ...func(*Options)) *ClientManager {
	options := Options{
		QPS:   50,
		Burst: 100,
		TTL:   1 * time.Minute,
	}
	for _, opt := range opts {
		opt(&options)
	}
	return &ClientManager{
		clusters: ttlcache.New(ttlcache.WithTTL[string, *Client](options.TTL)),
		ops:      options,
	}
}

type ClientManager struct {
	locker      sync.RWMutex
	clusters    *ttlcache.Cache[string, *Client]
	ops         Options
	localClient *Client
}

// GeteClient implements ClientManagerInterface.
func (c *ClientManager) GeteClient(clusterName string) (*Client, error) {
	if clusterName == LocalCluster {
		c.locker.RLock()
		localCluster := c.localClient
		c.locker.RUnlock()

		if localCluster != nil {
			return localCluster, nil
		}

		c.locker.Lock()
		defer c.locker.Unlock()
		_, err := c.buildClusterConfig(clusterName)
		if err != nil {
			return nil, err
		}

		return c.localClient, nil
	}

	v := c.clusters.Get(clusterName)
	if v != nil {
		return v.Value(), nil
	}
	c.locker.Lock()
	defer c.locker.Unlock()

	config, err := c.buildClusterConfig(clusterName)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, err
	}

	cs, err := clientset.NewForConfig(config)
	if err != nil {
		klog.ErrorS(err, "Failed to init clientset")
		return nil, err
	}

	k8sCli, err := client.New(config, client.Options{
		Scheme: gclient.NewSchema(),
	})
	if err != nil {
		klog.ErrorS(err, "Failed to init clientset")
		return nil, err
	}

	dc, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.ErrorS(err, "Failed to init clientset")
		return nil, err
	}
	cli := &Client{
		Interface: cs,
		Dynamic:   dc,
		Client:    k8sCli,
	}
	c.clusters.Set(clusterName, cli, ttlcache.DefaultTTL)
	return cli, nil
}

func (c *ClientManager) buildClusterConfig(clusterName string) (*rest.Config, error) {
	localConfig, err := utils.BuildLocalClusterConfig(c.ops.Kubeconfig)
	if err != nil {
		return nil, err
	}
	localClusterClient, err := kantaloupev1alpha1.NewForConfig(localConfig)
	if err != nil {
		return nil, err
	}
	localClient, err := clientset.NewForConfig(localConfig)
	if err != nil {
		return nil, err
	}
	k8sCli, err := client.New(localConfig, client.Options{
		Scheme: gclient.NewSchema(),
	})
	if err != nil {
		return nil, err
	}
	dc, err := dynamic.NewForConfig(localConfig)
	if err != nil {
		klog.ErrorS(err, "Failed to init clientset")
		return nil, err
	}
	c.localClient = &Client{
		Interface:                localClient,
		Client:                   k8sCli,
		Dynamic:                  dc,
		ClusterV1alpha1Interface: localClusterClient.ClusterV1alpha1(),
	}
	if clusterName == LocalCluster {
		return localConfig, nil
	}

	cluster, err := localClusterClient.ClusterV1alpha1().Clusters().Get(context.TODO(), clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return buildConfigFromSecretRef(localClient, cluster.Spec.SecretRef)
}

func buildConfigFromSecretRef(
	client clientset.Interface,
	ref *kantaloupeclusterv1alpha1.LocalSecretReference,
) (*rest.Config, error) {
	secret, err := client.CoreV1().Secrets(ref.Namespace).Get(context.TODO(), ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	kubeconfigBytes, ok := secret.Data["config"]
	if !ok {
		return nil, fmt.Errorf("the kubeconfig or data key 'kubeconfig' is not found,"+
			"please check the secret %s/%s", secret.Namespace, secret.Name)
	}
	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		return nil, err
	}
	return clientConfig.ClientConfig()
}
