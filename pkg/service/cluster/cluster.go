package cluster

import (
	"context"
	"fmt"
	"net/http"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	clustercrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/env"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/gclient"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/namespace"
)

type Service interface {
	ListClusters(ctx context.Context) ([]clustercrdv1alpha1.Cluster, error)
	CreateCluster(ctx context.Context, cluster *clustercrdv1alpha1.Cluster, kubeconfig string) (*clustercrdv1alpha1.Cluster, error)
	GetCluster(ctx context.Context, name string) (*clustercrdv1alpha1.Cluster, error)
	UpdateCluster(ctx context.Context, cluster *clustercrdv1alpha1.Cluster, kubeconfig string) (*clustercrdv1alpha1.Cluster, error)
	DeleteCluster(ctx context.Context, name string) error
	ValidateKubeconfig(ctx context.Context, kubeconfig string) (bool, error)
	ValidatePrometheusAddress(ctx context.Context, prometheusAddress string) (bool, error)
}

type service struct {
	clientManager engine.ClientManagerInterface
}

func NewService(clientManager engine.ClientManagerInterface) Service {
	return &service{
		clientManager: clientManager,
	}
}

// ListClusters lists clusters cr.
func (s *service) ListClusters(ctx context.Context) ([]clustercrdv1alpha1.Cluster, error) {
	localClient, err := s.clientManager.GeteClient(engine.LocalCluster)
	if err != nil {
		return nil, err
	}

	clusters, err := localClient.Clusters().List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to list clusters")
		return nil, err
	}
	return clusters.Items, nil
}

// CreateCluster creates a cluster cr in the gl	obal cluster.
func (s *service) CreateCluster(
	ctx context.Context,
	cluster *clustercrdv1alpha1.Cluster,
	kubeconfig string,
) (*clustercrdv1alpha1.Cluster, error) {
	localClient, err := s.clientManager.GeteClient(engine.LocalCluster)
	if err != nil {
		return nil, err
	}
	if err := monitoringv1.AddToScheme(localClient.Scheme()); err != nil {
		return nil, err
	}

	// Init clusterid from kube-system namespace uid.
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	k8sCli, err := client.New(restConfig, client.Options{
		Scheme: gclient.NewSchema(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	if err := monitoringv1.AddToScheme(k8sCli.Scheme()); err != nil {
		return nil, err
	}

	// TODO: duplicate field with `cluster.status.kubeSystemId`.
	ns := &corev1.Namespace{}
	err = k8sCli.Get(ctx, client.ObjectKey{Name: "kube-system"}, ns)
	if err != nil {
		return nil, err
	}
	cluster.Spec.ClusterId = string(ns.GetUID())

	// Validate if the Cluster already exists.
	if env.SkipCheckClusterKubesystemID.Get() == "false" {
		clusters := clustercrdv1alpha1.ClusterList{}
		if err := localClient.List(ctx, &clusters); err != nil {
			return nil, err
		}
		for _, c := range clusters.Items {
			if c.Spec.ClusterId == cluster.Spec.ClusterId {
				return nil, fmt.Errorf("cluster %s already exists", cluster.Name)
			}
		}
	}

	// create secret for cluster.
	clusterSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-secret", cluster.Name),
			Namespace: namespace.GetCurrentNamespaceOrDefault(),
			Labels:    map[string]string{constants.ClusterNameLableKey: cluster.Name},
		},
		Data: map[string][]byte{"config": []byte(kubeconfig)},
	}

	secret, err := s.createOrUpdateKubeconfig(ctx, clusterSecret)
	if err != nil {
		klog.ErrorS(err, "failed to create kubeconfig for cluster", "cluster", klog.KObj(cluster))
		return nil, err
	}

	// set the secret reference to cluster cr.
	cluster.Spec.SecretRef = &clustercrdv1alpha1.LocalSecretReference{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}

	if err := prepareCluster(ctx, cluster.Spec.Type, localClient, k8sCli); err != nil {
		klog.ErrorS(err, "failed to prepare cluster", "cluster", klog.KObj(cluster))
		return nil, err
	}

	created, err := localClient.Clusters().Create(ctx, cluster, metav1.CreateOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to create cluster", "cluster", klog.KObj(cluster))
		return nil, err
	}

	//  set ownerReference to secret.
	if err := controllerutil.SetOwnerReference(created, clusterSecret, gclient.NewSchema()); err != nil {
		return nil, err
	}

	if _, err := s.createOrUpdateKubeconfig(ctx, clusterSecret); err != nil {
		klog.ErrorS(err, "faild to set owner reference for cluster secret", "cluster", klog.KObj(cluster))
		deleterr := localClient.Clusters().Delete(ctx, cluster.Name, metav1.DeleteOptions{})
		if deleterr != nil {
			klog.ErrorS(deleterr, "failed to clear failed create cluster", "cluster", klog.KObj(cluster))
		}
		return nil, err
	}

	return cluster, nil
}

// Getcluster gets cluster cr from global cluster.
func (s *service) GetCluster(ctx context.Context, name string) (*clustercrdv1alpha1.Cluster, error) {
	localClient, err := s.clientManager.GeteClient(engine.LocalCluster)
	if err != nil {
		return nil, err
	}

	return localClient.Clusters().Get(ctx, name, metav1.GetOptions{})
}

func (s *service) UpdateCluster(ctx context.Context, cluster *clustercrdv1alpha1.Cluster, kubeconfig string) (*clustercrdv1alpha1.Cluster, error) {
	if kubeconfig != "" {
		clusterSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-secret", cluster.Name),
				Namespace: namespace.GetCurrentNamespaceOrDefault(),
				Labels:    map[string]string{constants.ClusterNameLableKey: cluster.Name},
			},
			Data: map[string][]byte{"config": []byte(kubeconfig)},
		}
		_, err := s.createOrUpdateKubeconfig(ctx, clusterSecret)
		if err != nil {
			klog.ErrorS(err, "failed to update kubeconfig for cluster", "cluster", klog.KObj(cluster))
			return nil, err
		}
	}
	localClient, err := s.clientManager.GeteClient(engine.LocalCluster)
	if err != nil {
		return nil, err
	}
	return localClient.Clusters().Update(ctx, cluster, metav1.UpdateOptions{})
}

// DeleteCluster deletes cluster cr in global cluster.
func (s *service) DeleteCluster(ctx context.Context, name string) error {
	localClient, err := s.clientManager.GeteClient(engine.LocalCluster)
	if err != nil {
		return err
	}
	return localClient.Clusters().Delete(ctx, name, metav1.DeleteOptions{})
}

// ValidateKubeconfig verfies the kubeconfig wherher is valid.
func (s *service) ValidateKubeconfig(_ context.Context, kubeconfig string) (bool, error) {
	// load the kubeconfig into config, and then use the config to create a
	// clientset to get the version of cluster. if it returns successfully,
	// it means it is valid.
	client, err := createClientFromKubeconfig([]byte(kubeconfig))
	if err != nil {
		return false, err
	}

	version, err := client.ServerVersion()
	if err != nil {
		return false, err
	}

	klog.V(2).InfoS("cluster version", "version", version.String())
	return true, nil
}

func (s *service) createOrUpdateKubeconfig(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error) {
	localClient, err := s.clientManager.GeteClient(engine.LocalCluster)
	if err != nil {
		return nil, err
	}

	current, err := localClient.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		// create a secret to store kubeconfig.
		return localClient.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	}

	if !equality.Semantic.DeepEqual(secret, current) {
		return localClient.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
	}
	return current, nil
}

// ValidatePrometheusAddress verifies whether the prometheus address is valid.
func (s *service) ValidatePrometheusAddress(ctx context.Context, prometheusAddress string) (bool, error) {
	// Create a new HTTP client with context timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create a new request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/-/healthy", prometheusAddress), nil)
	if err != nil {
		return false, err
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Check if the response status code is ok (200-299)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}

	return false, fmt.Errorf("prometheus health check failed with status code: %d", resp.StatusCode)
}

func createClientFromKubeconfig(kubeconfig []byte) (*clientset.Clientset, error) {
	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, err
	}
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	client, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func prepareCluster(ctx context.Context, clusterType string, localClient, newClient client.Client) error {
	names := []string{
		constants.HamiDevicePluginSMName,
		constants.HamiSchedulerSMName,
	}
	err := newClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "monitoring",
		},
	})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	switch clusterType {
	case "METAX":
		names = append(names, constants.MetaxSMTemplateName)
	case "NVIDIA":
		names = append(names, constants.NvidiaSMTemplateName)
	case "ASCEND":
		names = append(names, constants.AscendSMTemplateName)
	}
	for _, name := range names {
		if err := createServiceMonitor(ctx, localClient, newClient, name); err != nil {
			return err
		}
	}
	return nil
}

func createServiceMonitor(ctx context.Context, localClient, newClient client.Client, name string) error {
	template := &monitoringv1.ServiceMonitor{}
	if err := localClient.Get(ctx, client.ObjectKey{Namespace: "monitoring", Name: name}, template); err != nil {
		if apierrors.IsNotFound(err) {
			klog.InfoS("template not found, skip creating service monitor", "template", name)
			return nil
		}
		return fmt.Errorf("failed to get %s template: %w", name, err)
	}

	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      template.Name,
			Namespace: template.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, newClient, sm, func() error {
		sm.Labels = template.Labels
		sm.Spec = *template.Spec.DeepCopy()

		if sm.Labels != nil {
			if _, ok := sm.Labels["release"]; ok {
				sm.Labels["release"] = "prometheus"
			}
		}

		if len(sm.Spec.Endpoints) == 0 {
			return fmt.Errorf("template %s has no endpoints", name)
		}

		relabelings := sm.Spec.Endpoints[0].MetricRelabelConfigs
		if relabelings != nil {
			newRelabelings := make([]monitoringv1.RelabelConfig, 0, len(relabelings))
			for _, relabeling := range relabelings {
				if relabeling.TargetLabel != "cluster" {
					newRelabelings = append(newRelabelings, relabeling)
				}
			}
			sm.Spec.Endpoints[0].MetricRelabelConfigs = newRelabelings
		}
		return nil
	})

	return err
}
