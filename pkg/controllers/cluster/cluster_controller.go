package cluster

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	clustercrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/env"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/helper"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/informermanager"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/metrics"
)

const (
	// ClusterControllerName is the controller name that will be used when reporting events.
	ClusterControllerName = "cluster-controller"
	// ClusterControllerFinalizer is added to cluster to ensure cluster as well as the
	// execution space (namespace) is deleted before itself is deleted.
	ClusterControllerFinalizer = "kantaloupe.dynamic.io/cluster-controller"
	// RequeueAfter is the time after which the controller will requeue the request.
	RequeueAfter = 3 * time.Second
	// clusterResyncIntervalJitterFactor jitter factor for resyncInterval.
	clusterResyncIntervalJitterFactor = 0.5
	// CacheSyncTimeout refers to the time limit set on waiting for cache to sync.
	CacheSyncTimeout = 30 * time.Second
	// SyncGlobalClusterPeriod represents global cluster sync period.
	SyncGlobalClusterPeriod = 10 * time.Second
	// SyncLocalClusterPeriod represents prometheus federate member sync period.
	PrometheusConfigSyncPeriod = 15 * time.Second
	// CertificateValidityToleranceDuration defines the maximum tolerance of
	// global cluster kubeConfig certificate invalidation.
	CertificateValidityToleranceDuration = time.Hour * 24

	DefaultPrometheusAddress = "http://prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090"

	// ControllerName is the controller name that will be used when reporting events and metrics.
	clusterReady              = "ClusterReady"
	clusterHealthy            = "cluster is healthy and ready to accept workloads"
	clusterNotReady           = "ClusterNotReady"
	clusterUnhealthy          = "cluster is reachable but health endpoint responded without ok"
	clusterNotReachableReason = "ClusterNotReachable"
	clusterNotReachableMsg    = "cluster is not reachable"
	statusCollectionFailed    = "StatusCollectionFailed"

	metaxAnnotationKey        = "metax-tech.com/node-gpu-devices"
	nvidiaAnnotationKey       = "hami.io/node-nvidia-register"
	neuronDeviceKey           = "aws.amazon.com/neuron"
	ascendAnnotationPrefixKey = "hami.io/node-register-Ascend"
)

var kubeConfigCache sync.Map

// ContainerInstanceController is to sync ContainerInstance.
type Controller struct {
	client.Client        // used to operate ContainerInstance resources.
	InformerManager      informermanager.MultiClusterInformerManager
	EventRecorder        record.EventRecorder
	PredicateFunc        predicate.Predicate
	ClusterClientSetFunc func(string, client.Client, *utils.ClientOption) (*utils.ClusterClient, error)
	// ClusterClientOption holds the attributes that should be injected to a Kubernetes client.
	ClusterClientOption *utils.ClientOption
	// clusterConditionCache stores the condition status of each cluster.
	clusterConditionCache clusterConditionStore
	// ClusterSuccessThreshold is the duration of successes for the cluster to be considered healthy after recovery.
	ClusterSuccessThreshold metav1.Duration
	// ClusterFailureThreshold is the duration of failure for the cluster to be considered unhealthy.
	ClusterFailureThreshold metav1.Duration
	// ConcurrentClusterStatusSyncs is the number of cluster status that are allowed to sync concurrently.
	ConcurrentWorkSyncs int
	// ClusterStatusUpdateFrequency is the frequency that controller computes and report cluster status.
	ClusterStatusUpdateFrequency metav1.Duration
	// ClusterDebugMode indicates whether to enable debug mode for cluster.
	ClusterDebugMode bool
}

type tokenClaim struct {
	// WARNING: this JWT is not verified. Do not trust these claims.
	ExpTime int64 `json:"exp"`

	Kubernetes struct {
		Pod struct {
			UID string `json:"uid"`
		} `json:"pod"`
	} `json:"kubernetes.io"`
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *Controller) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.V(4).InfoS("Reconciling Cluster", "KObj", req.NamespacedName.String())

	start := time.Now()
	defer func() {
		metrics.RecordClusterStatusControllerReconcileDuration(req.Name, start)
		metrics.RecordClusterStatusControllerReconcileCount(req.Name)
	}()

	cluster := &clustercrdv1alpha1.Cluster{}
	if err := c.Client.Get(ctx, req.NamespacedName, cluster); err != nil {
		// The resource no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			c.InformerManager.Stop(req.Name)
			c.clusterConditionCache.delete(req.Name)
			return controllerruntime.Result{}, nil
		}
		return controllerruntime.Result{}, err
	}

	// remove the finalizer if the cluster is be deleted.
	if !cluster.DeletionTimestamp.IsZero() {
		// add loginc for deleting cluster
		c.InformerManager.Stop(req.Name)
		c.clusterConditionCache.delete(req.Name)
		if err := c.removeFinalizer(ctx, cluster); err != nil {
			klog.ErrorS(err, "failed to delete finalizer for cluster", "cluster", klog.KObj(cluster))
			return controllerruntime.Result{}, err
		}
		return controllerruntime.Result{}, nil
	}
	if err := c.ensureFinalizer(ctx, cluster); err != nil {
		klog.ErrorS(err, "faild to ensure finalizer for cluster", "cluster", klog.KObj(cluster))
		return controllerruntime.Result{}, err
	}

	if err := c.syncCluster(ctx, cluster); err != nil {
		klog.ErrorS(err, "faild to sync cluster", "cluster", klog.KObj(cluster))
		return controllerruntime.Result{}, err
	}

	return controllerruntime.Result{
		RequeueAfter: wait.Jitter(c.ClusterStatusUpdateFrequency.Duration, clusterResyncIntervalJitterFactor),
	}, nil
}

func (c *Controller) syncCluster(ctx context.Context, cluster *clustercrdv1alpha1.Cluster) error {
	start := time.Now()
	defer func() {
		metrics.RecordClusterStatus(cluster)
		metrics.RecordClusterSyncStatusDuration(cluster, start)
		metrics.RecordClusterSyncStatusCount(cluster)
	}()
	currentClusterStatus := *cluster.Status.DeepCopy()

	// create a ClusterClient for the given member cluster
	clusterClient, err := c.ClusterClientSetFunc(cluster.Name, c.Client, c.ClusterClientOption)
	if err != nil {
		klog.ErrorS(err, "Failed to create a ClusterClient for the given member cluster", "cluster", klog.KObj(cluster))
		return setStatusCollectionFailedCondition(ctx, c.Client, cluster, fmt.Sprintf("failed to create a ClusterClient: %v", err))
	}

	online, healthy := probeClusterHeart(ctx, clusterClient.KubeClient)
	observedReadyCondition := generateReadyCondition(online, healthy)
	readyCondition := c.clusterConditionCache.thresholdAdjustedReadyCondition(cluster, &observedReadyCondition)

	// cluster is offline after retry timeout, update cluster status immediately and return.
	if !online && readyCondition.Status != metav1.ConditionTrue {
		klog.V(2).InfoS("Cluster still offline after clusterFailureThreshold duration, ensuring offline is set.",
			"clusterFailureThreshold", c.ClusterFailureThreshold.Duration)
		return updateStatusCondition(ctx, c.Client, cluster, *readyCondition)
	}

	// skip collecting cluster status if not ready
	if online && healthy && readyCondition.Status == metav1.ConditionTrue {
		var conditions []metav1.Condition
		if err = c.setCurrentClusterStatus(ctx, clusterClient, cluster, &currentClusterStatus); err != nil {
			return err
		}
		conditions = append(conditions, *readyCondition)
		return c.updateStatusIfNeeded(ctx, cluster, currentClusterStatus, conditions...)
	}

	return nil
}

func (c *Controller) setCurrentClusterStatus(
	ctx context.Context,
	clusterClient *utils.ClusterClient, cluster *clustercrdv1alpha1.Cluster,
	currentClusterStatus *clustercrdv1alpha1.ClusterStatus,
) error {
	clusterVersion, err := getKubernetesVersion(clusterClient)
	if err != nil {
		klog.ErrorS(err, "failed to get Kubernetes version for cluster", "cluster", klog.KObj(cluster))
		return err
	}

	// get or create informer for pods and nodes in member cluster
	clusterInformerManager, err := c.buildInformerForCluster(clusterClient)
	if err != nil {
		klog.ErrorS(err, "failed to get or create informer for cluster", "cluster", klog.KObj(cluster))
		// in large-scale clusters, the timeout may occur.
		// if clusterInformerManager fails to be built, should be returned, otherwise, it may cause a nil pointer
		return err
	}

	kubeSystemID, err := getKubernetesKubeSystemID(clusterInformerManager)
	if err != nil {
		klog.ErrorS(err, "failed to get kubernetes system ID for Cluster", "cluster", klog.KObj(cluster))
		return err
	}

	nodes, err := listNodes(clusterInformerManager)
	if err != nil {
		klog.ErrorS(err, "Failed to list nodes for Cluster", "cluster", klog.KObj(cluster))
	}

	pods, err := listPods(clusterInformerManager)
	if err != nil {
		klog.ErrorS(err, "Failed to list pods for Cluster", "cluster", klog.KObj(cluster))
	}

	kts, err := clusterClient.GeneratedClient.KantaloupeflowV1alpha1().KantaloupeFlows(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "Failed to list kantaloupeflow for Cluster", "cluster", klog.KObj(cluster))
	}

	currentClusterStatus.KubernetesVersion = clusterVersion
	currentClusterStatus.KubeSystemID = kubeSystemID
	currentClusterStatus.NodeSummary = getNodeSummary(nodes)
	currentClusterStatus.ResourceSummary = getResourceSummary(nodes, pods)
	currentClusterStatus.KantaloupeflowSummary = &clustercrdv1alpha1.ResourceSummary{
		TotalNum: int32(len(kts.Items)),
		ReadyNum: helper.GetReadyKantaloupeflowNum(kts.Items),
	}
	currentClusterStatus.PodSetSummary = &clustercrdv1alpha1.ResourceSummary{
		TotalNum: int32(len(pods)), // #nosec G115
		ReadyNum: helper.GetReadyPodNum(pods),
	}

	return nil
}

func (c *Controller) ensureFinalizer(ctx context.Context, cluster *clustercrdv1alpha1.Cluster) error {
	if ctrlutil.AddFinalizer(cluster, ClusterControllerFinalizer) {
		return c.Client.Update(ctx, cluster)
	}
	return nil
}

func (c *Controller) removeFinalizer(ctx context.Context, cluster *clustercrdv1alpha1.Cluster) error {
	if err := c.DeleteScrapeConfig(ctx, cluster); err != nil {
		return err
	}
	// build member cluster client
	config, err := utils.ClusterKubeconfig(cluster.Name, c.Client, &utils.ClientOption{})
	if err != nil {
		return err
	}
	k8sClient, err := client.New(config, client.Options{})
	if err != nil {
		return err
	}
	monitoringv1.AddToScheme(k8sClient.Scheme())
	serviceMonitors := []string{
		constants.HamiDevicePluginSMName,
		constants.HamiSchedulerSMName,
		constants.MetaxSMTemplateName,
		constants.NvidiaSMTemplateName,
		constants.AscendSMTemplateName,
	}
	for _, name := range serviceMonitors {
		sm := &monitoringv1.ServiceMonitor{}
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "monitoring", Name: name}, sm)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		err = k8sClient.Delete(ctx, sm)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	if ctrlutil.RemoveFinalizer(cluster, ClusterControllerFinalizer) {
		return c.Client.Update(ctx, cluster)
	}
	return nil
}

// updateStatusIfNeeded calls updateStatus only if the status of the member cluster is
// not the same as the old status.
func (c *Controller) updateStatusIfNeeded(ctx context.Context, cluster *clustercrdv1alpha1.Cluster,
	currentClusterStatus clustercrdv1alpha1.ClusterStatus, conditions ...metav1.Condition,
) error {
	for _, condition := range conditions {
		meta.SetStatusCondition(&currentClusterStatus.Conditions, condition)
	}

	if !equality.Semantic.DeepEqual(cluster.Status, currentClusterStatus) {
		klog.V(4).InfoS("Start to update cluster status", "cluster", klog.KObj(cluster))
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err := utils.UpdateStatus(ctx, c.Client, cluster,
				func() error {
					cluster.Status.KubernetesVersion = currentClusterStatus.KubernetesVersion
					cluster.Status.KubeSystemID = currentClusterStatus.KubeSystemID
					cluster.Status.NodeSummary = currentClusterStatus.NodeSummary
					cluster.Status.PodSetSummary = currentClusterStatus.PodSetSummary
					cluster.Status.KantaloupeflowSummary = currentClusterStatus.KantaloupeflowSummary
					cluster.Status.ResourceSummary = currentClusterStatus.ResourceSummary

					// add others...

					for _, condition := range conditions {
						meta.SetStatusCondition(&cluster.Status.Conditions, condition)
					}

					return nil
				})
			return err
		})
		if err != nil {
			klog.ErrorS(err, "Failed to update health status of the member cluster", "cluste", klog.KObj(cluster))
			return err
		}
	}

	return nil
}

// buildInformerForCluster builds informer manager for cluster if it doesn't exist,
// then constructs informers for node and pod and start it. If the informer manager exist, return it.
func (c *Controller) buildInformerForCluster(clusterClient *utils.ClusterClient) (
	informermanager.SingleClusterInformerManager, error,
) {
	singleClusterInformerManager := c.InformerManager.GetSingleClusterManager(clusterClient.ClusterName)
	if singleClusterInformerManager == nil {
		singleClusterInformerManager = c.InformerManager.ForCluster(clusterClient.ClusterName, clusterClient.KubeClient, 0)
	}

	gvrs := []schema.GroupVersionResource{informermanager.NodeGVR, informermanager.PodGVR, informermanager.NamespaceGVR}

	// create the informer for pods and nodes
	allSynced := atomic.Bool{}
	allSynced.Store(true)
	wg := sync.WaitGroup{}
	for _, gvr := range gvrs {
		wg.Add(1)
		go func(gvr schema.GroupVersionResource) {
			defer wg.Done()
			if !singleClusterInformerManager.IsInformerSynced(gvr) {
				allSynced.Store(false)
				if _, err := singleClusterInformerManager.Lister(gvr); err != nil {
					klog.ErrorS(err, "Failed to get the lister for gvr", "gvr", gvr)
				}
			}
		}(gvr)
	}
	wg.Wait()
	if allSynced.Load() {
		return singleClusterInformerManager, nil
	}

	c.InformerManager.Start(clusterClient.ClusterName)

	if err := func(cluster string) error {
		synced := c.InformerManager.WaitForCacheSyncWithTimeout(cluster, CacheSyncTimeout)
		if synced == nil {
			return fmt.Errorf("no informerFactory for cluster %s exist", cluster)
		}
		for _, gvr := range gvrs {
			if !synced[gvr] {
				return fmt.Errorf("informer for %s hasn't synced", gvr)
			}
		}
		return nil
	}(clusterClient.ClusterName); err != nil {
		klog.ErrorS(err, "Failed to sync cache for cluster", "cluster", clusterClient.ClusterName)
		c.InformerManager.Stop(clusterClient.ClusterName)
		return nil, err
	}

	return singleClusterInformerManager, nil
}

// Start starts an asynchronous loop that monitors the status of cluster.
func (c *Controller) Start(ctx context.Context) error {
	klog.InfoS("Starting cluster health monitor")
	defer klog.InfoS("Shutting cluster health monitor")

	if err := monitoringv1alpha1.AddToScheme(c.Client.Scheme()); err != nil {
		return err
	}
	go wait.UntilWithContext(ctx, c.RunPrometheusConfigLoop, PrometheusConfigSyncPeriod)

	if c.ClusterDebugMode {
		klog.InfoS("Cluster debug mode is enabled")
		// Debug mode assumes that the kubeconfig will not change.
		home := homedir.HomeDir()
		config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
		if err != nil {
			klog.ErrorS(err, "Error get kubeconfig")
			return err
		}
		kubeConfig, err := ConvertToKubeConfig(config)
		if err != nil {
			klog.ErrorS(err, "Error ConvertToKubConfig")
			return err
		}
		go wait.UntilWithContext(ctx, func(ctx context.Context) {
			if err := c.EnsureGlobalClusterExist(ctx, kubeConfig); err != nil {
				klog.ErrorS(err, "Error ensure global cluster exist")
			}
		}, SyncGlobalClusterPeriod)

		<-ctx.Done()
		return nil
	}

	// Sync global cluster cr.
	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		klog.V(6).InfoS("begin sync global cluster cr.")
		// uses the current context in kubeConfig
		config, err := utils.BuildLocalClusterConfig("")
		if err != nil {
			klog.ErrorS(err, "Error buildConfigFromFlags")
			return
		}

		kubeConfig, err := ConvertToKubeConfigBySAToken(config)
		klog.V(6).InfoS("global cluster", "kubeConfig", kubeConfig)
		if err != nil {
			klog.ErrorS(err, "Error ConvertToKubConfigBySAToken")
			return
		}

		if err := c.EnsureGlobalClusterExist(ctx, kubeConfig); err != nil {
			klog.ErrorS(err, "Error sync global cluster")
		}
		klog.V(6).InfoS("sync global cluster cr succeed.")
	}, SyncGlobalClusterPeriod)

	<-ctx.Done()
	return nil
}

// EnsureGlobalClusterExist ensure global cluster crd already created.
func (c *Controller) EnsureGlobalClusterExist(ctx context.Context, kubeConfig string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-secret", constants.GlobalCluster),
			Namespace: helper.GetCurrentNSOrDefault(),
		},
		Data: map[string][]byte{
			"config": []byte(kubeConfig),
		},
	}

	if err := c.ensureSecretExist(ctx, secret); err != nil {
		return err
	}
	objectKey := types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}
	if err := c.Get(ctx, objectKey, secret); err != nil {
		return err
	}

	// Get the cluster uuid from kube-system namespace uuid.
	var uuid string
	ns := &corev1.Namespace{}
	if err := c.Client.Get(ctx, client.ObjectKey{Name: "kube-system"}, ns); err != nil {
		return err
	}
	uuid = string(ns.GetUID())

	// get gateway gatewayEndpoint from env.
	gatewayEndpoint := env.GatewayEndpoint.Get()
	if gatewayEndpoint == "" {
		gateway := gatewayv1.Gateway{}
		err := c.Get(ctx, client.ObjectKey{Namespace: helper.GetCurrentNSOrDefault(), Name: "kantaloupe"}, &gateway)
		if err != nil {
			klog.ErrorS(err, "failed to get gateway")
		}
		if len(gateway.Status.Addresses) > 0 {
			gatewayEndpoint = fmt.Sprintf("http://%s", gateway.Status.Addresses[0].Value)
		}
	}

	provider, clusterType, err := c.getClusterProviderAndType(ctx)
	if err != nil {
		klog.ErrorS(err, "failed to get cluster provider and type")
		return err
	}

	globalCluster := &clustercrdv1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       clustercrdv1alpha1.ClusterResourceKind,
			APIVersion: clustercrdv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.GlobalCluster,
		},
		Spec: clustercrdv1alpha1.ClusterSpec{
			ClusterId: uuid,
			Type:      clusterType,
			Provider:  provider,
			SecretRef: &clustercrdv1alpha1.LocalSecretReference{
				Namespace: secret.Namespace,
				Name:      secret.Name,
			},
			PrometheusAddress: DefaultPrometheusAddress,
			GatewayAddress:    gatewayEndpoint,
		},
	}
	return c.applyGlobalCluster(ctx, globalCluster)
}

// getClusterProviderAndType return cluster Provider and Type.
func (c *Controller) getClusterProviderAndType(ctx context.Context) (string, string, error) {
	// Get cluster type from node Annotations.
	clusterType := ""
	provider := constants.DefaultProvider

	var nodes corev1.NodeList
	if err := c.Client.List(ctx, &nodes); err != nil {
		return "", "", err
	}

	if len(nodes.Items) > 0 {
		providers := map[string]string{
			constants.GCPLabelKey: "GCP_GKE",
			constants.AWSLabelKey: "AWS_EKS",
		}
		for labelKey, prov := range providers {
			if _, ok := nodes.Items[0].Labels[labelKey]; ok {
				provider = prov
				break
			}
		}
	}

	for _, node := range nodes.Items {
		if _, exists := node.Annotations[metaxAnnotationKey]; exists {
			clusterType = "METAX"
		}
		if _, exists := node.Annotations[nvidiaAnnotationKey]; exists {
			clusterType = "NVIDIA"
		}
		if _, exists := node.Status.Allocatable[neuronDeviceKey]; exists {
			clusterType = "NEURON"
		}
		for key := range node.Annotations {
			if strings.HasPrefix(key, ascendAnnotationPrefixKey) {
				clusterType = "ASCEND"
				break
			}
		}
		if clusterType != "" {
			break
		}
	}

	return provider, clusterType, nil
}

// applyGlobalCluster make sure the global cluster is created (need to update if it exists).
// if the cluster not exists will create by given global cluster.
func (c *Controller) applyGlobalCluster(ctx context.Context, globalCluster *clustercrdv1alpha1.Cluster) error {
	currentCluster := globalCluster.DeepCopy()
	_, err := controllerruntime.CreateOrUpdate(ctx, c.Client, currentCluster, func() error {
		currentCluster.Labels = labels.Merge(currentCluster.Labels, globalCluster.Labels)
		currentCluster.Annotations = labels.Merge(currentCluster.Annotations, globalCluster.Annotations)
		currentCluster.Spec.APIEndpoint = globalCluster.Spec.APIEndpoint
		currentCluster.Spec.SecretRef = globalCluster.Spec.SecretRef
		currentCluster.Spec.Provider = globalCluster.Spec.Provider
		currentCluster.Spec.Type = globalCluster.Spec.Type
		currentCluster.Spec.ClusterId = globalCluster.Spec.ClusterId
		currentCluster.Spec.GatewayAddress = globalCluster.Spec.GatewayAddress
		currentCluster.Spec.PrometheusAddress = globalCluster.Spec.PrometheusAddress
		return nil
	})
	return err
}

// ensureSecretExist Make sure the secret exists, update it if it already exists, create it if it doesn't exist.
func (c *Controller) ensureSecretExist(ctx context.Context, newSecret *corev1.Secret) error {
	oldSecret := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Namespace: newSecret.Namespace, Name: newSecret.Name}, oldSecret)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		// Change secret does not exist before, create a new one.
		return c.Create(ctx, newSecret)
	}
	// add resourceVersion for newSecret in case the global cluster
	// reconcile frequently.
	newSecret.ResourceVersion = oldSecret.ResourceVersion

	// Check whether kubeConfig has changed.
	// If kubeConfig doesn't have any updates,return directly.
	return ProcessNewAndOldKubeConfig(newSecret.Data["config"], oldSecret.Data["config"], func(newSecretData []byte) error {
		newSecret.Data = map[string][]byte{"config": newSecretData}
		return c.Update(ctx, newSecret)
	})
}

// ProcessNewAndOldKubeConfig compare newSA.token and oldSA.token.
// if they are same, do nothing.
// if they are different, so update kubeConfig in secret ,and maybe the oldHost is egress proxy url which is updated by egress.Reconcile.
func ProcessNewAndOldKubeConfig(newKubeConfigBytes, oldKubeConfigBytes []byte, needUpdate func([]byte) error) error {
	newConfig, err := ParseKubeConfig(newKubeConfigBytes)
	if err != nil {
		// this should not happen
		return fmt.Errorf("parse new kubeConfig, err: %w", err)
	}
	oldConfig, err := ParseKubeConfig(oldKubeConfigBytes)
	if err != nil {
		// ignore, just log
		klog.V(5).ErrorS(err, "ignore parse old kubeConfig")
	}

	// For the third reconcile,
	// check whether the token is in the cache and validate the remaining effective time
	// for the token is greater than one day.
	// If the token is valid and the remaining effective time is greater than one day, skip the update directly
	token, _ := getTokenByKubeConfig(oldConfig)
	if exp, ok := kubeConfigCache.Load(token); ok && len(token) > 0 {
		claim := exp.(*tokenClaim)
		toleranceExpDate := metav1.NewTime(time.Now().Add(CertificateValidityToleranceDuration))
		expDate := metav1.Unix(claim.ExpTime, 0)

		newToken, _ := getTokenByKubeConfig(newConfig)
		newClaim, ok := parseTokenClaimByToken(newToken)

		if ok && newClaim.Kubernetes.Pod.UID == claim.Kubernetes.Pod.UID && toleranceExpDate.Before(&expDate) {
			// the old kubeConfig is still in effect, do nothing
			klog.V(6).InfoS("global cluster kubeConfig is still in effect, use the old kubeConfig in cache")
			return nil
		}
	}

	// For the second reconcile,
	// compare whether the new and old tokens are the same,
	// if they are the same, save the token to the cache.
	if EqualKubeConfigWithToken(newConfig, oldConfig) {
		// these tokens are same, do nothing
		claim, ok := parseTokenClaimByToken(token)
		if ok {
			kubeConfigCache.Store(token, claim)
		}
		return nil
	}

	// For the first reconcile,
	// after the pod restarted for the first time,
	// we need to verify whether the original kubeConfig is invalid.
	// If it is valid, save it to the cache.
	if checkKubeConfigValid(oldKubeConfigBytes) {
		// the old kubeConfig is valid, save it to cache
		claim, ok := parseTokenClaimByToken(token)
		if ok {
			kubeConfigCache.Store(token, claim)
		}
		return nil
	}

	MergeKubeConfigWithServerAddr(newConfig, oldConfig)
	newSecretData, err := WriteKubeConfig(newConfig)
	if err != nil {
		return err
	}
	return needUpdate(newSecretData)
}

// ParseKubeConfig Parse kubeConfig string to struct value.
func ParseKubeConfig(kubeConfig []byte) (*clientcmdapi.Config, error) {
	if len(kubeConfig) == 0 {
		return nil, fmt.Errorf("parse nil kubeConfig")
	}
	return clientcmd.Load(kubeConfig)
}

func parseTokenClaimByToken(tokenData string) (*tokenClaim, bool) {
	parts := strings.Split(tokenData, ".")
	if len(parts) != 3 {
		return nil, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	claim := &tokenClaim{}
	if err := json.Unmarshal(payload, &claim); err != nil {
		return nil, false
	}
	return claim, true
}

func getTokenByKubeConfig(config *clientcmdapi.Config) (string, error) {
	if config == nil {
		return "", fmt.Errorf("KubeConfig is Empty")
	}

	if auth, ok := config.AuthInfos[config.Contexts[config.CurrentContext].AuthInfo]; ok && auth != nil {
		return strings.TrimSpace(auth.Token), nil
	}
	return "", fmt.Errorf("CurrentContext %s can not get authInfos", config.CurrentContext)
}

// EqualKubeConfigWithToken if newSAToken and oldSAToken are same ,so return true.
func EqualKubeConfigWithToken(newConfig, oldConfig *clientcmdapi.Config) bool {
	if newConfig == nil || oldConfig == nil {
		return false
	}
	if newConfig.AuthInfos[newConfig.Contexts[newConfig.CurrentContext].AuthInfo].Token == "" {
		klog.InfoS("Detected no token in kubeconfig, are you debugging?")
		return true
	}
	newToken, err := getTokenByKubeConfig(newConfig)
	if err != nil {
		// this should not happen
		klog.ErrorS(err, "Parse New KubeConfig")
		return false
	}
	oldToken, err := getTokenByKubeConfig(oldConfig)
	if err != nil {
		klog.ErrorS(err, "Parse Old KubeConfig")
		return false
	}
	return newToken == oldToken
}

// checkKubeConfigValid checks whether the provided kubeConfig is valid
// by get the kube-system ns.
func checkKubeConfigValid(bytes []byte) bool {
	clientConfig, err := clientcmd.NewClientConfigFromBytes(bytes)
	if err != nil {
		return false
	}

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return false
	}
	helper.MakeConfigSkipTLS(config)

	cs, err := clientset.NewForConfig(config)
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	_, err = cs.CoreV1().Namespaces().Get(ctx, metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to check global cluster kubeConfig valid")
	}
	return err == nil
}

// MergeKubeConfigWithServerAddr replace newConfig.Host with oldConfig.Host if values are not empty.
// keep the host of oldConfig ,because maybe the old host is egress proxy addr.
func MergeKubeConfigWithServerAddr(newConfig, oldConfig *clientcmdapi.Config) {
	if newConfig == nil || oldConfig == nil ||
		newConfig.Clusters[newConfig.CurrentContext] == nil || oldConfig.Clusters[oldConfig.CurrentContext] == nil {
		return
	}
	if newConfig.Clusters[newConfig.CurrentContext] != nil && oldConfig.Clusters[oldConfig.CurrentContext] != nil {
		newConfig.Clusters[newConfig.CurrentContext].Server = oldConfig.Clusters[oldConfig.CurrentContext].Server
	}
}

// WriteKubeConfig generate kubeConfig string from struct value.
func WriteKubeConfig(config *clientcmdapi.Config) ([]byte, error) {
	if config == nil {
		return nil, fmt.Errorf("config nil")
	}
	return clientcmd.Write(*config)
}

// SetupWithManager creates a controller and register to controller manager.
func (c *Controller) SetupWithManager(mgr controllerruntime.Manager) error {
	mgr.GetLogger().V(4).Info("cluster status controller concurrent reconciles", "MaxConcurrentReconciles", c.ConcurrentWorkSyncs)

	c.clusterConditionCache = clusterConditionStore{
		successThreshold: c.ClusterSuccessThreshold.Duration,
		failureThreshold: c.ClusterFailureThreshold.Duration,
	}

	return utilerrors.NewAggregate([]error{
		controllerruntime.NewControllerManagedBy(mgr).
			Owns(&corev1.Secret{}).
			For(&clustercrdv1alpha1.Cluster{}, builder.WithPredicates(c.PredicateFunc)).
			WithOptions(controller.Options{
				MaxConcurrentReconciles: c.ConcurrentWorkSyncs,
				LogConstructor: func(request *reconcile.Request) logr.Logger {
					logger := mgr.GetLogger().WithName(ClusterControllerName)
					if request != nil {
						return logger.WithValues("cluster", request.Name)
					}
					return logger
				},
			}).Complete(c),
		mgr.Add(c),
	})
}

// probeClusterHeart to probe wether the cluster is online and healthy.
// the first return param represents whether it is online, and the second
// param represents whether it is healthy.
func probeClusterHeart(ctx context.Context, clientset clientset.Interface) (bool, bool) {
	logger := klog.FromContext(ctx)
	healthStatus, err := healthEndpointCheck(ctx, clientset, "/readyz")
	if err != nil && healthStatus == http.StatusNotFound {
		// do health check with api endpoint if the readyz endpoint is not installed in member cluster
		healthStatus, err = healthEndpointCheck(ctx, clientset, "/api")
	}

	if err != nil {
		logger.Error(err, "Failed to do cluster health check for cluster")
		return false, false
	}

	if healthStatus != http.StatusOK {
		logger.Error(errors.New("http status isn't ok"), "Member cluster isn't healthy")
		return true, false
	}
	return true, true
}

func getKubernetesVersion(clusterClient *utils.ClusterClient) (string, error) {
	clusterVersion, err := clusterClient.KubeClient.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}

	return clusterVersion.GitVersion, nil
}

func getKubernetesKubeSystemID(informerManager informermanager.SingleClusterInformerManager) (string, error) {
	nsInterface, err := informerManager.Lister(informermanager.NamespaceGVR)
	if err != nil {
		return "", err
	}
	nsLister, ok := nsInterface.(listercorev1.NamespaceLister)
	if !ok {
		return "", fmt.Errorf("failed to convert interface to nsLister")
	}

	ns, err := nsLister.Get("kube-system")
	if err != nil {
		return "", err
	}
	return string(ns.GetUID()), nil
}

func healthEndpointCheck(ctx context.Context, client clientset.Interface, path string) (int, error) {
	var healthStatus int
	resp := client.Discovery().RESTClient().Get().AbsPath(path).Do(ctx).StatusCode(&healthStatus)
	return healthStatus, resp.Error()
}

func generateReadyCondition(online, healthy bool) metav1.Condition {
	if !online {
		return utils.NewCondition(clustercrdv1alpha1.ClusterConditionReady, clusterNotReachableReason,
			clusterNotReachableMsg, metav1.ConditionFalse)
	}
	if !healthy {
		return utils.NewCondition(clustercrdv1alpha1.ClusterConditionReady, clusterNotReady, clusterUnhealthy, metav1.ConditionFalse)
	}

	return utils.NewCondition(clustercrdv1alpha1.ClusterConditionReady, clusterReady, clusterHealthy, metav1.ConditionTrue)
}

func setStatusCollectionFailedCondition(ctx context.Context,
	c client.Client, cluster *clustercrdv1alpha1.Cluster, message string,
) error {
	readyCondition := utils.NewCondition(clustercrdv1alpha1.ClusterConditionReady,
		statusCollectionFailed, message, metav1.ConditionFalse)

	return updateStatusCondition(ctx, c, cluster, readyCondition)
}

func updateStatusCondition(ctx context.Context, c client.Client,
	cluster *clustercrdv1alpha1.Cluster, conditions ...metav1.Condition,
) error {
	klog.V(4).InfoS("Start to update cluster status condition", "cluster", klog.KObj(cluster))
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := utils.UpdateStatus(ctx, c, cluster,
			func() error {
				for _, condition := range conditions {
					meta.SetStatusCondition(&cluster.Status.Conditions, condition)
				}
				return nil
			})
		return err
	})
	if err != nil {
		klog.ErrorS(err, "Failed to update status condition of the member cluster", "cluster", klog.KObj(cluster))
		return err
	}
	return nil
}

// listPods returns the Pod list from the informerManager cache.
func listPods(informerManager informermanager.SingleClusterInformerManager) ([]*corev1.Pod, error) {
	podInterface, err := informerManager.Lister(informermanager.PodGVR)
	if err != nil {
		return nil, err
	}

	podLister, ok := podInterface.(listercorev1.PodLister)
	if !ok {
		return nil, fmt.Errorf("failed to convert interface to PodLister")
	}

	return podLister.List(labels.Everything())
}

// listNodes returns the Node list from the informerManager cache.
func listNodes(informerManager informermanager.SingleClusterInformerManager) ([]*corev1.Node, error) {
	nodeInterface, err := informerManager.Lister(informermanager.NodeGVR)
	if err != nil {
		return nil, err
	}

	nodeLister, ok := nodeInterface.(listercorev1.NodeLister)
	if !ok {
		return nil, fmt.Errorf("failed to convert interface to NodeLister")
	}

	return nodeLister.List(labels.Everything())
}

func getNodeSummary(nodes []*corev1.Node) *clustercrdv1alpha1.ResourceSummary {
	totalNum := len(nodes)
	readyNum := 0

	for _, node := range nodes {
		if helper.NodeReady(node) {
			readyNum++
		}
	}

	nodeSummary := &clustercrdv1alpha1.ResourceSummary{}
	nodeSummary.TotalNum = int32(totalNum) // #nosec G115
	nodeSummary.ReadyNum = int32(readyNum) // #nosec G115

	return nodeSummary
}

func getResourceSummary(nodes []*corev1.Node, pods []*corev1.Pod) *clustercrdv1alpha1.ClusterResourceSummary {
	allocatable := getClusterAllocatable(nodes)
	allocating := getAllocatingResource(pods)
	allocated := getAllocatedResource(pods)

	resourceSummary := &clustercrdv1alpha1.ClusterResourceSummary{}
	resourceSummary.Allocatable = allocatable
	resourceSummary.Allocating = allocating
	resourceSummary.Allocated = allocated

	return resourceSummary
}

func getClusterAllocatable(nodeList []*corev1.Node) corev1.ResourceList {
	allocatable := make(corev1.ResourceList)
	var gpuTotal int // the number of gpu
	var gpuMemoryTotal int
	for _, node := range nodeList {
		for key, val := range node.Status.Allocatable {
			tmpCap, ok := allocatable[key]
			if ok {
				tmpCap.Add(val)
			} else {
				tmpCap = val
			}
			allocatable[key] = tmpCap
		}

		if val, ok := node.Annotations[constants.HamiRegisterAnonationKey]; ok {
			gpuInfos := strings.Split(val, ":")

			for _, info := range gpuInfos {
				res := strings.Split(info, ",")
				if len(res) == 7 {
					gpuMemoryStr := res[2]
					gpuMeory, err := strconv.Atoi(gpuMemoryStr)
					if err != nil {
						klog.ErrorS(err, "failed parse gpu memory if node", "node", node.GetName())
					}
					gpuMemoryTotal += gpuMeory
					gpuTotal++
				}
			}
		}
	}

	// TODO: the number of gpu
	allocatable["nvidia.com/gpu.count"] = resource.MustParse(strconv.Itoa(gpuTotal))
	allocatable["nvidia.com/gpu-memory.count"] = resource.MustParse(strconv.Itoa(gpuMemoryTotal))

	return allocatable
}

func getAllocatingResource(podList []*corev1.Pod) corev1.ResourceList {
	allocating := utils.EmptyResource()
	podNum := int64(0)
	for _, pod := range podList {
		if len(pod.Spec.NodeName) == 0 {
			allocating.AddPodRequest(&pod.Spec)
			podNum++
		}
	}
	allocating.AddResourcePods(podNum)
	return allocating.ResourceList()
}

func getAllocatedResource(podList []*corev1.Pod) corev1.ResourceList {
	allocated := utils.EmptyResource()
	podNum := int64(0)
	for _, pod := range podList {
		// When the phase of a pod is Succeeded or Failed, kube-scheduler would not consider its resource occupation.
		if len(pod.Spec.NodeName) != 0 && pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			allocated.AddPodRequest(&pod.Spec)
			podNum++
		}
	}
	allocated.AddResourcePods(podNum)
	return allocated.ResourceList()
}

// ConvertToKubeConfigBySAToken convert service account token to kubeConfig.
func ConvertToKubeConfigBySAToken(config *rest.Config) (string, error) {
	if config == nil {
		return "", fmt.Errorf("KubeConfig is Empty")
	}
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", err
	}
	rest.LoadTLSFiles(config)
	template := constants.KubeConfigDefaultTokenTemplate
	template = strings.TrimSpace(template)

	return fmt.Sprintf(template, base64.StdEncoding.EncodeToString(config.CAData), config.Host, token), nil
}

func ConvertToKubeConfig(config *rest.Config) (string, error) {
	if config == nil {
		return "", fmt.Errorf("KubeConfig is Empty")
	}
	rest.LoadTLSFiles(config)
	template := constants.KubeConfigTemplate
	template = strings.TrimSpace(template)

	return fmt.Sprintf(template, base64.StdEncoding.EncodeToString(config.CAData), config.Host,
		base64.StdEncoding.EncodeToString(config.CertData), base64.StdEncoding.EncodeToString(config.KeyData)), nil
}
