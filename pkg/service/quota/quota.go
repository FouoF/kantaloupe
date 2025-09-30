package quota

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/service/monitoring"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/namespace"
)

// Service defines the interface for quota operations.
type Service interface {
	// ListQuotas lists resource quotas with filtering options.
	ListQuotas(ctx context.Context, namespace, cluster string) ([]*corev1.ResourceQuota, error)

	// CreateQuota creates a new resource quota.
	CreateQuota(ctx context.Context, name, namespace, cluster string, hard map[string]string) (*corev1.ResourceQuota, error)

	// GetQuota gets a resource quota.
	GetQuota(ctx context.Context, name, namespace, cluster string) (*corev1.ResourceQuota, error)

	// UpdateQuota updates an existing resource quota.
	UpdateQuota(ctx context.Context, name, namespace, cluster string, hard map[string]string) (*corev1.ResourceQuota, error)

	// DeleteQuota deletes a resource quota by name.
	DeleteQuota(ctx context.Context, name, namespace, cluster string) error
}

// service implements the Service interface.
type service struct {
	clientManager     engine.ClientManagerInterface
	monitoringService monitoring.Service
}

// NewService creates a new quota service.
func NewService(clientManager engine.ClientManagerInterface, client engine.PrometheusInterface) Service {
	return &service{
		clientManager:     clientManager,
		monitoringService: monitoring.NewService(client),
	}
}

// getClient returns a Kubernetes client for the specified cluster.
func (s *service) getClient(cluster string) (*engine.Client, error) {
	target := cluster
	if target == "" {
		target = engine.LocalCluster
	}
	return s.clientManager.GeteClient(target)
}

// getNamespace returns the namespace, defaulting if necessary.
func getNamespace(ns string) string {
	if ns == "" {
		return namespace.GetCurrentNamespaceOrDefault()
	}
	return ns
}

// ListQuotas lists resource quotas with optional filtering.
func (s *service) ListQuotas(ctx context.Context, namespace, cluster string) ([]*corev1.ResourceQuota, error) {
	client, err := s.getClient(cluster)
	if err != nil {
		klog.ErrorS(err, "failed to get Kubernetes client")
		return nil, err
	}

	if namespace == constants.SelectAll {
		namespace = metav1.NamespaceAll
	}
	quotaList, err := client.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range quotaList.Items {
		s.patchGPUQuota(ctx, cluster, &quotaList.Items[i])
	}
	return utils.SliceToPointerSlice(quotaList.Items), nil
}

func (s *service) GetQuota(ctx context.Context, name, namespace, cluster string) (*corev1.ResourceQuota, error) {
	client, err := s.getClient(cluster)
	if err != nil {
		klog.ErrorS(err, "failed to get Kubernetes client", "cluster", cluster)
		return nil, err
	}

	quota, err := client.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to get resource quota", "name", name, "namespace", namespace)
		return nil, err
	}
	s.patchGPUQuota(ctx, cluster, quota)
	return quota, nil
}

// CreateQuota creates a new resource quota.
func (s *service) CreateQuota(
	ctx context.Context,
	name, ns, cluster string,
	hard map[string]string,
) (*corev1.ResourceQuota, error) {
	client, err := s.getClient(cluster)
	if err != nil {
		klog.ErrorS(err, "failed to get Kubernetes client", "cluster", cluster)
		return nil, err
	}

	// Build label selector for managed resources
	labelSelector := constants.ManagedByLabelKey + "=" + constants.ManagedByLabelValue

	// List resource quotas with label filtering
	var quotaList *corev1.ResourceQuotaList
	var listErr error

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	quotaList, listErr = client.CoreV1().ResourceQuotas(metav1.NamespaceAll).List(ctx, listOptions)
	if listErr != nil {
		klog.ErrorS(listErr, "failed to list resource quotas", "labelSelector", labelSelector)
		return nil, listErr
	}

	for i := range quotaList.Items {
		quota := &quotaList.Items[i]
		if quota.Namespace == ns {
			return nil, errors.New("the namespace already exists")
		}
	}

	namespace := getNamespace(ns)

	// Prepare the resource quota
	quota, err := prepareResourceQuota(name, namespace, hard)
	if err != nil {
		klog.ErrorS(err, "failed to prepare resource quota", "name", name)
		return nil, err
	}

	// Add management labels
	addQuotaLabels(quota)

	// Create the resource quota
	createdQuota, err := client.CoreV1().ResourceQuotas(namespace).Create(ctx, quota, metav1.CreateOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to create resource quota", "name", name, "namespace", namespace)
		return nil, err
	}

	klog.V(4).InfoS("created resource quota", "name", name, "namespace", namespace)
	return createdQuota, nil
}

// UpdateQuota updates an existing resource quota.
func (s *service) UpdateQuota(
	ctx context.Context,
	name, ns, cluster string,
	hard map[string]string,
) (*corev1.ResourceQuota, error) {
	client, err := s.getClient(cluster)
	if err != nil {
		klog.ErrorS(err, "failed to get Kubernetes client", "cluster", cluster)
		return nil, err
	}

	namespace := getNamespace(ns)

	// Get existing quota
	existingQuota, err := client.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to get existing resource quota", "name", name, "namespace", namespace)
		return nil, err
	}

	// Prepare updated quota
	updatedQuota, err := prepareResourceQuota(name, namespace, hard)
	if err != nil {
		klog.ErrorS(err, "failed to prepare updated resource quota", "name", name)
		return nil, err
	}

	// Preserve metadata
	updatedQuota.ResourceVersion = existingQuota.ResourceVersion
	if existingQuota.Labels != nil {
		for k, v := range existingQuota.Labels {
			if updatedQuota.Labels == nil {
				updatedQuota.Labels = make(map[string]string)
			}
			updatedQuota.Labels[k] = v
		}
	}
	if existingQuota.Annotations != nil {
		updatedQuota.Annotations = existingQuota.Annotations
	}

	// Update management labels
	addQuotaLabels(updatedQuota)

	// Update the resource quota
	updatedQuota, err = client.CoreV1().ResourceQuotas(namespace).Update(ctx, updatedQuota, metav1.UpdateOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to update resource quota", "name", name, "namespace", namespace)
		return nil, err
	}

	klog.V(4).InfoS("updated resource quota", "name", name, "namespace", namespace)
	return updatedQuota, nil
}

// DeleteQuota deletes a resource quota by name.
func (s *service) DeleteQuota(ctx context.Context, name, namespace, cluster string) error {
	client, err := s.getClient(cluster)
	if err != nil {
		klog.ErrorS(err, "failed to get Kubernetes client")
		return err
	}

	namespace = getNamespace(namespace)

	// Verify it's a quota we manage
	quota, err := client.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			klog.V(4).InfoS("resource quota not found, skipping deletion", "name", name, "namespace", namespace)
			return nil
		}
		klog.ErrorS(err, "failed to get resource quota for deletion", "name", name, "namespace", namespace)
		return err
	}

	// Verify it's managed by kantaloupe
	if quota.Labels == nil || quota.Labels[constants.ManagedByLabelKey] != constants.ManagedByLabelValue {
		klog.V(4).InfoS("resource quota not managed by kantaloupe, skipping deletion", "name", name, "namespace", namespace)
		return nil
	}

	err = client.CoreV1().ResourceQuotas(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		klog.ErrorS(err, "failed to delete resource quota", "name", name, "namespace", namespace)
		return err
	}

	klog.V(4).InfoS("deleted resource quota", "name", name, "namespace", namespace)
	return nil
}

// prepareResourceQuota creates a ResourceQuota object based on provided hard limits.
func prepareResourceQuota(name, namespace string, hard map[string]string) (*corev1.ResourceQuota, error) {
	hardResources, err := parseResourceList(hard)
	if err != nil {
		return nil, err
	}

	return &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: hardResources,
		},
	}, nil
}

// parseResourceList converts a string map to a ResourceList.
func parseResourceList(resources map[string]string) (corev1.ResourceList, error) {
	resourceList := corev1.ResourceList{}
	for k, v := range resources {
		quantity, err := resource.ParseQuantity(v)
		if err != nil {
			klog.ErrorS(err, "failed to parse resource quantity", "key", k, "value", v)
			return nil, err
		}
		resourceList[corev1.ResourceName(k)] = quantity
	}
	return resourceList, nil
}

// addQuotaLabels adds management labels to a resource quota.
func addQuotaLabels(quota *corev1.ResourceQuota) {
	if quota.Labels == nil {
		quota.Labels = make(map[string]string)
	}

	// Mark as managed by kantaloupe
	quota.Labels[constants.ManagedByLabelKey] = constants.ManagedByLabelValue
}

func (s *service) patchGPUQuota(ctx context.Context, cluster string, quota *corev1.ResourceQuota) error {
	vec, err := s.monitoringService.QueryVector(ctx, fmt.Sprintf(`QuotaUsed{cluster="%s", quotanamespace="%s"}`, cluster, quota.Namespace))
	if err != nil {
		return err
	}

	// Ensure quota.Status.Used is initialized to prevent a panic.
	if quota.Status.Used == nil {
		quota.Status.Used = make(corev1.ResourceList)
	}

	for _, val := range vec {
		qName, ok := val.Metric["quotaName"]
		if !ok || qName == "" {
			continue
		}
		quota.Status.Used[corev1.ResourceName(fmt.Sprintf("requests.%s", corev1.ResourceName(qName)))] = resource.MustParse(fmt.Sprint(val.Value))
	}
	return nil
}
