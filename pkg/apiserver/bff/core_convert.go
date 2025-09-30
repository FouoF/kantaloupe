package bff

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/copier"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/dynamia-ai/kantaloupe/api/core/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/api/types"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
)

func convertAccessMode(mode string) ([]corev1.PersistentVolumeAccessMode, error) {
	switch strings.ToUpper(mode) {
	case "RWO", "READWRITONCE":
		return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, nil
	case "ROX", "READONLYMANY":
		return []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany}, nil
	case "RWX", "READWRITEMANY":
		return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}, nil
	default:
		return nil, fmt.Errorf("unsupported AccessMode: %s", mode)
	}
}

func convertObjectMeta(objectMeta metav1.ObjectMeta) *types.ObjectMeta {
	objectMeta.SetManagedFields(nil)
	metadata := &types.ObjectMeta{
		Name:              objectMeta.Name,
		ResourceVersion:   objectMeta.ResourceVersion,
		Namespace:         objectMeta.Namespace,
		Uid:               string(objectMeta.UID),
		CreationTimestamp: objectMeta.CreationTimestamp.Unix(),
		Labels:            objectMeta.Labels,
		Annotations:       objectMeta.Annotations,
		OwnerReferences:   convertOwnRef(objectMeta.OwnerReferences),
	}

	if objectMeta.DeletionTimestamp != nil {
		metadata.DeletionTimestamp = objectMeta.DeletionTimestamp.Unix()
	}
	return metadata
}

func convertOwnRef(ownerRefs []metav1.OwnerReference) []*types.OwnerReference {
	res := make([]*types.OwnerReference, 0, len(ownerRefs))
	for _, ref := range ownerRefs {
		res = append(res, &types.OwnerReference{
			Uid:  string(ref.UID),
			Name: ref.Name,
			Kind: ref.Kind,
		})
	}

	return res
}

func convertProto2PersistentVolume(req *corev1alpha1.PersistentVolume) (*corev1.PersistentVolume, error) {
	accessModes := make([]corev1.PersistentVolumeAccessMode, 0)
	for _, mode := range req.GetSpec().GetAccessModes() {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode.String()))
	}
	quantity, err := resource.ParseQuantity(req.GetSpec().GetCapacity())
	if err != nil {
		return nil, err
	}
	volumeMode := corev1.PersistentVolumeMode(req.GetSpec().GetVolumeMode().String())
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.GetMetadata().GetName(),
			Namespace: req.GetMetadata().GetNamespace(),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: quantity,
			},
			AccessModes:                   accessModes,
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimPolicy(req.GetSpec().GetPersistentVolumeReclaimPolicy().String()),
			VolumeMode:                    &volumeMode,
			PersistentVolumeSource:        corev1.PersistentVolumeSource{},
		},
	}
	pvSource := req.GetSpec().GetPersistentVolumeSource()
	switch {
	case pvSource.GetHostPath() != nil:
		hostPathType := pvSource.GetHostPath().GetType()
		pv.Spec.HostPath = &corev1.HostPathVolumeSource{
			Path: pvSource.GetHostPath().GetPath(),
			Type: (*corev1.HostPathType)(&hostPathType),
		}
	case pvSource.GetNfs() != nil:
		pv.Spec.NFS = &corev1.NFSVolumeSource{
			Server:   pvSource.GetNfs().GetServer(),
			Path:     pvSource.GetNfs().GetPath(),
			ReadOnly: pvSource.GetNfs().GetReadOnly(),
		}
	case pvSource.GetLocal() != nil:
		pv.Spec.Local = &corev1.LocalVolumeSource{
			Path:   pvSource.GetLocal().GetPath(),
			FSType: &pvSource.GetLocal().FsType,
		}
	default:
		return nil, fmt.Errorf("unsupported volume source type")
	}
	return pv, nil
}

func convertPersistentVolumeSpec(spec corev1.PersistentVolumeSpec) (*corev1alpha1.PersistentVolumeSpec, error) {
	persistentVolumeSpec := &corev1alpha1.PersistentVolumeSpec{}
	if err := copier.Copy(persistentVolumeSpec, spec); err != nil {
		return nil, err
	}
	quantity := spec.Capacity[corev1.ResourceStorage]
	persistentVolumeSpec.Capacity = quantity.String()
	persistentVolumeSpec.PersistentVolumeReclaimPolicy = corev1alpha1.PersistentVolumeReclaimPolicy(
		corev1alpha1.PersistentVolumeReclaimPolicy_value[string(spec.PersistentVolumeReclaimPolicy)],
	)
	for _, mode := range spec.AccessModes {
		persistentVolumeSpec.AccessModes = append(persistentVolumeSpec.AccessModes,
			corev1alpha1.PersistentVolumeAccessMode(
				corev1alpha1.PersistentVolumeAccessMode_value[string(mode)]),
		)
	}
	if spec.VolumeMode != nil {
		persistentVolumeSpec.VolumeMode = corev1alpha1.PersistentVolumeMode(
			corev1alpha1.PersistentVolumeMode_value[string(*spec.VolumeMode)],
		)
	}
	persistentVolumeSpec.PersistentVolumeSource = convertPersistentVolumeSource(spec.PersistentVolumeSource)
	return persistentVolumeSpec, nil
}

func convertPersistentVolumeSource(pvSource corev1.PersistentVolumeSource) *corev1alpha1.PersistentVolumeSource {
	ret := &corev1alpha1.PersistentVolumeSource{}
	if pvSource.HostPath != nil {
		ret.HostPath = &corev1alpha1.HostPathVolumeSource{}
		ret.HostPath.Path = pvSource.HostPath.Path
		ret.HostPath.Type = string(*pvSource.HostPath.Type)
	}
	if pvSource.NFS != nil {
		ret.Nfs = &corev1alpha1.NFSVolumeSource{}
		ret.Nfs.Server = pvSource.NFS.Server
		ret.Nfs.Path = pvSource.NFS.Path
		ret.Nfs.ReadOnly = pvSource.NFS.ReadOnly
	}
	if pvSource.Local != nil {
		ret.Local = &corev1alpha1.LocalVolumeSource{}
		ret.Local.Path = pvSource.Local.Path
		if pvSource.Local.FSType != nil {
			ret.Local.FsType = *pvSource.Local.FSType
		}
	}
	return ret
}

func convertPersistentVolumeStatus(persistentVolume corev1.PersistentVolume) *corev1alpha1.PersistentVolumeStatus {
	persistentVolumeStatus := persistentVolume.Status
	result := &corev1alpha1.PersistentVolumeStatus{
		Phase: corev1alpha1.Phase(
			corev1alpha1.Phase_value[string(persistentVolumeStatus.Phase)],
		),
		Message: persistentVolumeStatus.Message,
		Reason:  persistentVolumeStatus.Reason,
	}
	return result
}

func convertSecret2Proto(secret corev1.Secret) *corev1alpha1.Secret {
	secretData := make(map[string]string)
	for k := range secret.Data {
		secretData[k] = base64.StdEncoding.EncodeToString(secret.Data[k])
	}

	return &corev1alpha1.Secret{
		Metadata: convertObjectMeta(secret.ObjectMeta),
		Type:     string(secret.Type),
	}
}

func convertProto2Secret(secret *corev1alpha1.Secret) (*corev1.Secret, error) {
	data := make(map[string][]byte)
	for k := range secret.GetData() {
		decoded, err := base64.StdEncoding.DecodeString(secret.GetData()[k])
		if err != nil {
			return nil, err
		}
		data[k] = decoded
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.GetMetadata().GetName(),
			Namespace: secret.GetMetadata().GetNamespace(),
		},
		Data: data,
		Type: corev1.SecretType(secret.GetType()),
	}, nil
}

func convertSecrets2Proto(secrets []*corev1.Secret) []*corev1alpha1.Secret {
	result := make([]*corev1alpha1.Secret, 0, len(secrets))
	for _, secret := range secrets {
		result = append(result, convertSecret2Proto(*secret))
	}
	return result
}

// func convertNamespaces(namespaces []corev1.Namespace) []*corev1alpha1.Namespace {
// 	result := make([]*corev1alpha1.Namespace, len(namespaces))
// 	for index := range namespaces {
// 		result[index] = convertNamespace(namespaces[index])
// 	}
// 	return result
// }

// func convertNamespace(namespace corev1.Namespace) *corev1alpha1.Namespace {
// 	res := &corev1alpha1.Namespace{
// 		Metadata: convertObjectMeta(namespace.ObjectMeta),
// 		Spec:     &corev1alpha1.NamespaceSpec{},
// 		Status:   convertNamespaceStatus(namespace.Status, namespace.Labels),
// 	}
// 	for _, v := range namespace.Spec.Finalizers {
// 		res.Spec.Finalizers = append(res.Spec.Finalizers, string(v))
// 	}
// 	return res
// }

// func convertNamespaceStatus(status corev1.NamespaceStatus, labels map[string]string) *corev1alpha1.NamespaceStatus {
// 	namespacestatus := &corev1alpha1.NamespaceStatus{
// 		Phase:              convertNamespacePhase(status.Phase),
// 		Conditions:         convertNamespaceCondition(status.Conditions),
// 		PodSecurityEnabled: isPodSecurityEnabled(labels),
// 	}
// 	return namespacestatus
// }

// The label "pod-security.kubernetes.io/<MODE>-version: <VERSION>" alone will not take effect, therefore only "pod-security.kubernetes.io/<MODE>: <LEVEL>" decides if pod security is enabled.
// func isPodSecurityEnabled(labelsSet map[string]string) bool {
// 	if labels.Set(labelsSet).Has(constants.PodSecurityLabelEnforce) || labels.Set(labelsSet).Has(constants.PodSecurityLabelAudit) || labels.Set(labelsSet).Has(constants.PodSecurityLabelWarn) {
// 		return true
// 	}
// 	return false
// }

// func convertNamespaceCondition(conditions []corev1.NamespaceCondition) []*types.Condition {
// 	result := make([]*types.Condition, len(conditions))
// 	for index, v := range conditions {
// 		result[index] = &types.Condition{
// 			Message:            v.Message,
// 			Reason:             v.Reason,
// 			LastTransitionTime: timeutil.Format(v.LastTransitionTime.Time),
// 			Status:             string(v.Status),
// 			Type:               string(v.Type),
// 		}
// 	}
// 	return result
// }

// func convertNamespacePhase(status corev1.NamespacePhase) corev1alpha1.NamespacePhase {
// 	switch status {
// 	case corev1.NamespaceActive:
// 		return corev1alpha1.NamespacePhase_Active
// 	case corev1.NamespaceTerminating:
// 		return corev1alpha1.NamespacePhase_Terminating
// 	}
// 	return corev1alpha1.NamespacePhase_NAMESPACE_PHASE_UNSPECIFIED
// }

func convertEvents(events []*corev1.Event) []*corev1alpha1.Event {
	result := make([]*corev1alpha1.Event, 0, len(events))
	for _, event := range events {
		lastTime := event.LastTimestamp
		if lastTime.Unix() < 0 {
			lastTime = metav1.Time(event.EventTime)
		}
		result = append(result, &corev1alpha1.Event{
			InvolvedObject: &corev1alpha1.ObjectReference{
				Kind:      event.InvolvedObject.Kind,
				Name:      event.InvolvedObject.Name,
				Namespace: event.InvolvedObject.Namespace,
			},
			Reason:         event.Reason,
			Message:        event.Message,
			Source:         &corev1alpha1.EventSource{Component: event.Source.Component, Host: event.Source.Host},
			LastTimestamp:  lastTime.Unix(),
			FirstTimestamp: event.FirstTimestamp.Unix(),
			Type:           corev1alpha1.EventType(corev1alpha1.EventType_value[event.Type]),
		})
	}
	return result
}

func convertNode2Proto(node *corev1.Node, metric *nodeMetric) *corev1alpha1.Node {
	convertedNode := &corev1alpha1.Node{
		Metadata: &types.ObjectMeta{},
		Spec:     &corev1alpha1.NodeSpec{},
		Status:   &corev1alpha1.NodeStatus{},
	}
	convertedNode.Metadata = convertObjectMeta(node.ObjectMeta)

	convertedNode.Spec.PodCIDR = node.Spec.PodCIDR
	convertedNode.Spec.Unschedulable = node.Spec.Unschedulable
	convertedNode.Status.Status = fetchNodeCondition(node.Status, node.Spec)
	convertedNode.Status.Roles = convertToRoles(node.ObjectMeta)
	convertedNode.Status.SystemInfo = &corev1alpha1.NodeSystemInfo{
		KernelVersion:           node.Status.NodeInfo.KernelVersion,
		OsImage:                 node.Status.NodeInfo.OSImage,
		ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
		KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
		Architecture:            node.Status.NodeInfo.Architecture,
	}

	for _, addr := range node.Status.Addresses {
		addressType := addr.Type
		if addressType != corev1.NodeHostName {
			address := []*corev1alpha1.NodeAddress{
				{
					Type:    string(addressType),
					Address: addr.Address,
				},
			}
			convertedNode.Status.Addresses = append(convertedNode.Status.Addresses, address...)
		}
	}

	for _, taint := range node.Spec.Taints {
		taint := []*corev1alpha1.Taint{
			{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: convertTaintEffect(taint.Effect),
			},
		}
		convertedNode.Spec.Taints = append(convertedNode.Spec.Taints, taint...)
	}

	if metric != nil {
		convertedNode.Status.GpuCount = metric.gpuCount
		convertedNode.Status.CpuCapacity = metric.cpuCapacity
		convertedNode.Status.CpuUsage = metric.cpuUsage
		convertedNode.Status.CpuAllocated = metric.cpuAllocated

		convertedNode.Status.MemoryCapacity = metric.memoryCapacity
		convertedNode.Status.MemoryUsage = metric.memoryUsage
		convertedNode.Status.MemoryAllocated = metric.memoryAllocated

		convertedNode.Status.GpuCoreTotal = metric.gpuCoreTotal
		convertedNode.Status.GpuCoreAllocated = metric.gpuCoreAllocated
		convertedNode.Status.GpuCoreUsage = metric.gpuCoreUsage
		convertedNode.Status.GpuMemoryTotal = metric.gpuMemoryTotal
		convertedNode.Status.GpuMemoryAllocatable = metric.gpuMemoryAllocatable
		convertedNode.Status.GpuMemoryAllocated = metric.gpuMemoryAllocated
		convertedNode.Status.GpuMemoryUsage = metric.gpuMemoryUsage
	}

	// convertedNode.Status.CpuCapacity = node.Status.Capacity.Cpu().MilliValue()
	// convertedNode.Status.MemoryCapacity = node.Status.Capacity.Memory().Value()

	return convertedNode
}

func convertTaintEffect(effect corev1.TaintEffect) corev1alpha1.TaintEffect {
	var rsp corev1alpha1.TaintEffect
	switch effect {
	case corev1.TaintEffectNoSchedule:
		rsp = corev1alpha1.TaintEffect_NoSchedule
	case corev1.TaintEffectPreferNoSchedule:
		rsp = corev1alpha1.TaintEffect_PreferNoSchedule
	case corev1.TaintEffectNoExecute:
		rsp = corev1alpha1.TaintEffect_NoExecute
	}
	return rsp
}

func fetchNodeCondition(nodeStatus corev1.NodeStatus, nodeSpec corev1.NodeSpec) *corev1alpha1.NodeStatus_Status {
	res := &corev1alpha1.NodeStatus_Status{}
	conditions := []*corev1alpha1.NodeCondition{}
	for _, condition := range nodeStatus.Conditions {
		nodeCondition := &corev1alpha1.NodeCondition{
			Type:            string(condition.Type),
			Status:          types.ConditionStatus(types.ConditionStatus_value[string(condition.Status)]),
			Reason:          condition.Reason,
			Message:         condition.Message,
			UpdateTimestamp: condition.LastHeartbeatTime.Format(time.RFC3339),
		}
		conditions = append(conditions, nodeCondition)
	}

	conditions = append(conditions, &corev1alpha1.NodeCondition{
		Type:            constants.NodeSchedulingDisabled,
		Status:          types.ConditionStatus(types.ConditionStatus_value[cases.Title(language.Und).String(strconv.FormatBool(nodeSpec.Unschedulable))]),
		UpdateTimestamp: time.Now().Format(time.RFC3339),
		Message:         fmt.Sprintf("Node scheduling disabled: %t", nodeSpec.Unschedulable),
	})
	res.Phase = GetNodePhase(nodeStatus.Conditions)
	res.Conditions = conditions

	return res
}

func GetNodePhase(conditions []corev1.NodeCondition) corev1alpha1.NodePhase {
	var res corev1alpha1.NodePhase
	for _, condition := range conditions {
		if condition.Type == corev1.NodeReady {
			switch status := condition.Status; status {
			case corev1.ConditionTrue:
				res = corev1alpha1.NodePhase_Ready
			case corev1.ConditionFalse:
				res = corev1alpha1.NodePhase_Not_Ready
			default:
				res = corev1alpha1.NodePhase_Unknown
			}
		}
	}
	return res
}

func convertToRoles(objectMeta metav1.ObjectMeta) []corev1alpha1.Role {
	var roles []corev1alpha1.Role
	masterLabelKeys := []string{constants.ControllerLabelKey, constants.MasterLabelKey}
	isMaster := func(labels map[string]string) bool {
		for _, labelKey := range masterLabelKeys {
			_, ok := labels[labelKey]
			if ok {
				return true
			}
		}
		return false
	}
	if isMaster(objectMeta.Labels) {
		roles = append(roles, corev1alpha1.Role_CONTROL_PLANE)
	}

	if _, ok := objectMeta.Labels[constants.WorkerLabelKey]; ok {
		roles = append(roles, corev1alpha1.Role_WORKER)
	}
	if len(roles) == 0 {
		roles = append(roles, corev1alpha1.Role_WORKER)
	}
	return roles
}

func convertConfigMap2Proto(configMap corev1.ConfigMap) *corev1alpha1.ConfigMap {
	result := &corev1alpha1.ConfigMap{
		Metadata: convertObjectMeta(configMap.ObjectMeta),
		// Immutable:  false,
		Data:       configMap.Data,
		BinaryData: configMap.BinaryData,
	}

	if configMap.Immutable != nil {
		result.Immutable = *configMap.Immutable
	}

	return result
}

func taintEffectConvert(effect corev1alpha1.TaintEffect) corev1.TaintEffect {
	var rsp corev1.TaintEffect
	switch effect {
	case corev1alpha1.TaintEffect_NoSchedule:
		rsp = corev1.TaintEffectNoSchedule
	case corev1alpha1.TaintEffect_PreferNoSchedule:
		rsp = corev1.TaintEffectPreferNoSchedule
	case corev1alpha1.TaintEffect_NoExecute:
		rsp = corev1.TaintEffectNoExecute
	}
	return rsp
}
