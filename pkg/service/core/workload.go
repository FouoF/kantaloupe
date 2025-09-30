package core

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	storagev1alpha1 "github.com/dynamia-ai/kantaloupe/api/storage/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
)

type Service interface {
	ListPersistentVolumes(ctx context.Context, cluster string) ([]*corev1.PersistentVolume, error)
	GetPersistentVolume(ctx context.Context, cluster, name string) (*corev1.PersistentVolume, error)
	CreatePersistentVolume(ctx context.Context, cluster string, persistentVolume *corev1.PersistentVolume) (*corev1.PersistentVolume, error)
	UpdatePersistentVolume(ctx context.Context, cluster string, persistentVolume *corev1.PersistentVolume) (*corev1.PersistentVolume, error)
	DeletePersistentVolume(ctx context.Context, cluster, name string) error

	UpdateSecret(ctx context.Context, cluster string, secret *corev1.Secret) (*corev1.Secret, error)
	GetSecret(ctx context.Context, cluster, namespace, name string) (*corev1.Secret, error)
	DeleteSecret(ctx context.Context, cluster, namespace, name string) error
	CreateSecret(ctx context.Context, cluster, namespace string, secret *corev1.Secret) (*corev1.Secret, error)
	ListSecrets(ctx context.Context, cluster, namespace string) ([]*corev1.Secret, error)

	ListStorageClasses(ctx context.Context, cluster string) ([]*storagev1.StorageClass, error)

	ListNamespaces(ctx context.Context, cluster string) ([]*corev1.Namespace, error)

	CreatePersistentVolumeClaim(ctx context.Context, cluster, namespace string, pvc *corev1.PersistentVolumeClaim, storageType storagev1alpha1.StorageType) (*corev1.PersistentVolumeClaim, error)
	DeletePersistentVolumeClaim(ctx context.Context, cluster, namespace, name string) error
	ListPersistentVolumeClaims(ctx context.Context, cluster, namespace string, managerFlag bool) ([]*corev1.PersistentVolumeClaim, error)
	AddPersistentVolumeClaimLabel(ctx context.Context, cluster, namespace, name string) (*corev1.PersistentVolumeClaim, error)

	ListEventsByDeployment(ctx context.Context, cluster, namespace, deployment string) ([]*corev1.Event, error)
	ListEventsByPod(ctx context.Context, cluster, namespace, pod string) ([]*corev1.Event, error)
	ListEvents(ctx context.Context, cluster, namespace string) ([]*corev1.Event, error)

	ListPods(ctx context.Context, cluster, namespace, labelSelector string) ([]*corev1.Pod, error)
	GetPod(ctx context.Context, cluster, namespace, pod string) (*corev1.Pod, error)

	ListNodes(ctx context.Context, cluster string) ([]*corev1.Node, error)
	GetNode(ctx context.Context, cluster, name string) (*corev1.Node, error)
	PutNodeLabels(ctx context.Context, clusterName, name string, labels map[string]string) (map[string]string, error)
	PutNodeTaints(ctx context.Context, clusterName, name string, taints []*corev1.Taint) ([]corev1.Taint, error)
	UpdateNodeAnnotations(ctx context.Context, cluster, node string, annotations map[string]string) (map[string]string, error)
	UnScheduleNode(ctx context.Context, cluster, node string, unschedulable bool) (*corev1.Node, error)

	GetConfigMap(ctx context.Context, cluster, namespace, name string) (*corev1.ConfigMap, error)
	UpdateConfigMap(ctx context.Context, cluster, namespace string, configmap *corev1.ConfigMap) (*corev1.ConfigMap, error)
}

var _ Service = new(service)

func NewService(clientManager engine.ClientManagerInterface) Service {
	return &service{
		clientManager: clientManager,
	}
}

type service struct {
	clientManager engine.ClientManagerInterface
}

func (s *service) ListPersistentVolumes(ctx context.Context, cluster string) ([]*corev1.PersistentVolume, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	pvs, err := client.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to list pvs")
		return nil, err
	}

	return utils.SliceToPointerSlice(pvs.Items), nil
}

func (s *service) GetPersistentVolume(ctx context.Context, cluster, name string) (*corev1.PersistentVolume, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	return client.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
}

func (s *service) CreatePersistentVolume(ctx context.Context, cluster string, persistentVolume *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	return client.CoreV1().PersistentVolumes().Create(ctx, persistentVolume, metav1.CreateOptions{})
}

func (s *service) UpdatePersistentVolume(ctx context.Context, cluster string, persistentvolume *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	return client.CoreV1().PersistentVolumes().Update(ctx, persistentvolume, metav1.UpdateOptions{})
}

func (s *service) DeletePersistentVolume(ctx context.Context, cluster, name string) error {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return err
	}
	return client.CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{})
}

func (s *service) CreateSecret(ctx context.Context, cluster, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	return client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
}

func (s *service) GetSecret(ctx context.Context, cluster, namespace, name string) (*corev1.Secret, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (s *service) UpdateSecret(ctx context.Context, cluster string, secret *corev1.Secret) (*corev1.Secret, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	return client.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
}

func (s *service) DeleteSecret(ctx context.Context, cluster, namespace, name string) error {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return err
	}
	return client.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (s *service) ListSecrets(ctx context.Context, cluster, namespace string) ([]*corev1.Secret, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	secrets, err := client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to list secrets")
		return nil, err
	}
	return utils.SliceToPointerSlice(secrets.Items), nil
}

func (s *service) ListPods(ctx context.Context, cluster, namespace, labelSelector string) ([]*corev1.Pod, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}

	return utils.SliceToPointerSlice(pods.Items), nil
}

func (s *service) ListStorageClasses(ctx context.Context, cluster string) ([]*storagev1.StorageClass, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	storageClasses, err := client.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to list storage classes")
		return nil, err
	}

	return utils.SliceToPointerSlice(storageClasses.Items), nil
}

func (s *service) ListNamespaces(ctx context.Context, cluster string) ([]*corev1.Namespace, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to list namespaces")
		return nil, err
	}

	return utils.SliceToPointerSlice(namespaces.Items), nil
}

func (s *service) CreatePersistentVolumeClaim(ctx context.Context, cluster, namespace string, pvc *corev1.PersistentVolumeClaim, storageType storagev1alpha1.StorageType) (*corev1.PersistentVolumeClaim, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	if pvc.Labels == nil {
		pvc.Labels = make(map[string]string)
	}
	if pvc.Annotations == nil {
		pvc.Annotations = make(map[string]string)
	}
	pvc.Annotations[constants.StorageTypeKey] = storageType.String()
	pvc.Labels[constants.ManagedByLabelKey] = constants.ManagedByLabelValue
	return client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
}

func (s *service) DeletePersistentVolumeClaim(ctx context.Context, cluster, namespace, name string) error {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return err
	}
	return client.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (s *service) ListPersistentVolumeClaims(ctx context.Context, cluster, namespace string, managerFlag bool) ([]*corev1.PersistentVolumeClaim, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	labelSelector := ""
	if managerFlag {
		labelSelector = constants.ManagedByLabelKey + "=" + constants.ManagedByLabelValue
	}
	options := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if namespace == constants.SelectAll {
		namespace = corev1.NamespaceAll
	}

	pvcs, err := client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, options)
	if err != nil {
		klog.ErrorS(err, "failed to list pvcs")
		return nil, err
	}

	return utils.SliceToPointerSlice(pvcs.Items), nil
}

func (s *service) AddPersistentVolumeClaimLabel(ctx context.Context, cluster, namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if pvc.Labels == nil {
		pvc.Labels = make(map[string]string)
	}
	pvc.Labels[constants.ManagedByLabelKey] = constants.ManagedByLabelValue
	pv, err := client.CoreV1().PersistentVolumes().Get(ctx, pvc.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if pv.Annotations == nil {
		pv.Annotations = make(map[string]string)
	}
	pvc.Annotations[constants.StorageTypeKey] = getStorageType(pv).String()

	pvc, err = client.CoreV1().PersistentVolumeClaims(namespace).Update(ctx, pvc, metav1.UpdateOptions{})
	return pvc, err
}

func getStorageType(pv *corev1.PersistentVolume) storagev1alpha1.StorageType {
	storageType := storagev1alpha1.StorageType_PVC
	if pv.Spec.PersistentVolumeSource.NFS != nil {
		storageType = storagev1alpha1.StorageType_NFS
	} else if pv.Spec.PersistentVolumeSource.Local != nil {
		storageType = storagev1alpha1.StorageType_LocalPV
	}
	return storageType
}

func (s *service) ListEvents(ctx context.Context, cluster, namespace string) ([]*corev1.Event, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	events, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "failed to list events")
		return nil, err
	}

	return utils.SliceToPointerSlice(events.Items), nil
}

func (s *service) ListEventsByDeployment(ctx context.Context, cluster, namespace, deployment string) ([]*corev1.Event, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		klog.ErrorS(err, "failed to get client")
		return nil, err
	}
	deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	events, err := client.CoreV1().Events(deploy.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.kind=Deployment,involvedObject.name=" + deploy.Name,
	})
	if err != nil {
		klog.ErrorS(err, "failed to list events")
		return nil, err
	}

	return utils.SliceToPointerSlice(events.Items), nil
}

func (s *service) ListEventsByPod(ctx context.Context, cluster, namespace, pod string) ([]*corev1.Event, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		klog.ErrorS(err, "failed to get client")
		return nil, err
	}
	po, err := s.GetPod(ctx, cluster, namespace, pod)
	if err != nil {
		return nil, err
	}
	events, err := client.CoreV1().Events(po.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.kind=Pod,involvedObject.name=" + po.Name,
	})
	if err != nil {
		klog.ErrorS(err, "failed to list events")
		return nil, err
	}

	return utils.SliceToPointerSlice(events.Items), nil
}

func (s *service) GetPod(ctx context.Context, cluster, namespace, pod string) (*corev1.Pod, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	return client.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
}

func (s *service) ListNodes(ctx context.Context, cluster string) ([]*corev1.Node, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return utils.SliceToPointerSlice(nodes.Items), nil
}

func (s *service) PutNodeLabels(ctx context.Context, cluster, name string, labels map[string]string) (map[string]string, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	loadBytes, _ := utils.LabelsConvert(labels)
	node, err := client.CoreV1().Nodes().Patch(ctx, name, types.JSONPatchType, loadBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}
	return node.Labels, nil
}

func (s *service) PutNodeTaints(ctx context.Context, cluster, name string, taints []*corev1.Taint) ([]corev1.Taint, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	loadBytes, _ := utils.NodeTaintsConvert(taints)
	node, err := client.CoreV1().Nodes().Patch(ctx, name, types.JSONPatchType, loadBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}

	return node.Spec.Taints, nil
}

func (s *service) UpdateNodeAnnotations(ctx context.Context, cluster, node string, annotations map[string]string) (map[string]string, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	loadBytes, _ := utils.AnnotationsConvert(annotations)
	patchNode, err := client.CoreV1().Nodes().Patch(ctx, node, types.JSONPatchType, loadBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}
	return patchNode.Annotations, nil
}

func (s *service) UnScheduleNode(ctx context.Context, cluster, node string, unschedulable bool) (*corev1.Node, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	loadBytes, _ := utils.NodeScheduleConvert(unschedulable)
	patchNode, err := client.CoreV1().Nodes().Patch(ctx, node, types.JSONPatchType, loadBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}
	return patchNode, nil
}

func (s *service) UpdateNode(ctx context.Context, cluster string, node *corev1.Node) (*corev1.Node, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}
	return client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
}

func (s *service) GetNode(ctx context.Context, cluster, name string) (*corev1.Node, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	node, err := client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (s *service) GetConfigMap(ctx context.Context, cluster, namespace, name string) (*corev1.ConfigMap, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	configmap, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return configmap, nil
}

func (s *service) UpdateConfigMap(ctx context.Context, cluster, namespace string, configmap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	client, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	older, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, configmap.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// update latest resourceVersion to the configmap.
	configmap.ResourceVersion = older.ResourceVersion
	newer, err := client.CoreV1().ConfigMaps(namespace).Update(ctx, configmap, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return newer, nil
}
