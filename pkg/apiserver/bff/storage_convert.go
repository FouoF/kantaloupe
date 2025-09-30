package bff

import (
	"errors"
	"fmt"

	"k8s.io/klog/v2"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/dynamia-ai/kantaloupe/api/core/v1alpha1"
	storagev1alpha1 "github.com/dynamia-ai/kantaloupe/api/storage/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
)

func convertStorageClass(storageClass *storagev1.StorageClass) *storagev1alpha1.StorageClass {
	result := &storagev1alpha1.StorageClass{
		Metadata:         convertObjectMeta(storageClass.ObjectMeta),
		Provisioner:      storageClass.Provisioner,
		StorageClassName: storageClass.Name,
		MountOptions:     storageClass.MountOptions,
		Parameters:       storageClass.Parameters,
	}

	if storageClass.AllowVolumeExpansion != nil {
		result.AllowVolumeExpansion = *storageClass.AllowVolumeExpansion
	}

	if storageClass.ReclaimPolicy != nil {
		result.ReclaimPolicy = storagev1alpha1.StorageClass_ReclaimPolicy(storagev1alpha1.StorageClass_ReclaimPolicy_value[string(*storageClass.ReclaimPolicy)])
	}
	if storageClass.VolumeBindingMode != nil {
		result.VolumeBindingMode = storagev1alpha1.StorageClass_VolumeBindingMode(storagev1alpha1.StorageClass_VolumeBindingMode_value[string(*storageClass.VolumeBindingMode)])
	}

	return result
}

func convertStorageClasses(storageClasses []*storagev1.StorageClass) []*storagev1alpha1.StorageClass {
	result := make([]*storagev1alpha1.StorageClass, 0, len(storageClasses))
	for _, storageClass := range storageClasses {
		result = append(result, convertStorageClass(storageClass))
	}
	return result
}

func (h *StorageHandler) convertRequestToNFSPV(req *storagev1alpha1.CreateStorageRequest) (*corev1.PersistentVolume, error) {
	if req.GetNfsServer() == "" || req.GetDataPath() == "" {
		return nil, errors.New("NFS PV requires NFSServer and DataPath")
	}
	if req.GetStorageName() == "" || req.GetStorageSize() == "" {
		return nil, errors.New("PV requires StorageName and StorageSize")
	}

	accessModes, err := convertAccessMode(req.GetAccessMode())
	if err != nil {
		return nil, err
	}

	capacity, err := resource.ParseQuantity(req.GetStorageSize())
	if err != nil {
		return nil, fmt.Errorf("invalid StorageSize: %w", err)
	}

	storageClassName := req.GetNamespace() + "-" + req.GetStorageName()

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.GetStorageName(),
			Labels: map[string]string{
				constants.PVCTypeLabelKey: storagev1alpha1.StorageType_NFS.String(),
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: capacity,
			},
			AccessModes:                   accessModes,
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              storageClassName,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				NFS: &corev1.NFSVolumeSource{
					Server: req.GetNfsServer(),
					Path:   req.GetDataPath(),
				},
			},
		},
	}
	return pv, nil
}

func (h *StorageHandler) convertRequestToLocalPV(req *storagev1alpha1.CreateStorageRequest) (*corev1.PersistentVolume, error) {
	if req == nil {
		return nil, errors.New("CreateStorageRequest cannot be nil")
	}
	if req.GetLocalPath() == "" || req.GetNodeName() == "" {
		return nil, errors.New("local PV requires LocalPath and NodeName")
	}
	if req.GetStorageName() == "" || req.GetStorageSize() == "" {
		return nil, errors.New("PV requires StorageName and StorageSize")
	}

	accessModes, err := convertAccessMode(req.GetAccessMode())
	if err != nil {
		return nil, err
	}

	capacity, err := resource.ParseQuantity(req.GetStorageSize())
	if err != nil {
		return nil, fmt.Errorf("invalid StorageSize: %w", err)
	}

	storageClassName := "local-storage"

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.GetStorageName(),
			Labels: map[string]string{
				constants.PVCTypeLabelKey: storagev1alpha1.StorageType_LocalPV.String(),
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: capacity,
			},
			AccessModes:                   accessModes,
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              storageClassName,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				Local: &corev1.LocalVolumeSource{
					Path: req.GetLocalPath(),
				},
			},
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/hostname",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{req.GetNodeName()},
								},
							},
						},
					},
				},
			},
		},
	}
	return pv, nil
}

func (h *StorageHandler) convertRequestToPVC(req *storagev1alpha1.CreateStorageRequest, pvStorageClassName string) (*corev1.PersistentVolumeClaim, error) {
	if req == nil {
		return nil, errors.New("CreateStorageRequest cannot be nil")
	}
	accessModes, err := convertAccessMode(req.GetAccessMode())
	if err != nil {
		return nil, err
	}

	capacity, err := resource.ParseQuantity(req.GetStorageSize())
	if err != nil {
		return nil, fmt.Errorf("invalid StorageSize: %w", err)
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.GetStorageName(),
			Namespace: req.GetNamespace(),
			Labels: map[string]string{
				constants.PVCTypeLabelKey: req.GetStorageType().String(),
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: capacity,
				},
			},
		},
	}

	if pvStorageClassName != "" {
		pvc.Spec.StorageClassName = &pvStorageClassName
	} else if req.GetStorageClassName() != "" {
		scName := req.GetStorageClassName()
		pvc.Spec.StorageClassName = &scName
	} else {
		pvc.Spec.StorageClassName = nil
	}

	return pvc, nil
}

func convertPersistentVolumes2Proto(persistentVolumes []*corev1.PersistentVolume) []*corev1alpha1.PersistentVolume {
	result := make([]*corev1alpha1.PersistentVolume, 0, len(persistentVolumes))
	for _, persistentVolume := range persistentVolumes {
		result = append(result, convertPersistentVolume2Proto(persistentVolume))
	}
	return result
}

func convertPersistentVolume2Proto(persistentVolume *corev1.PersistentVolume) *corev1alpha1.PersistentVolume {
	spec, err := convertPersistentVolumeSpec(persistentVolume.Spec)
	if err != nil {
		spec = &corev1alpha1.PersistentVolumeSpec{}
	}

	return &corev1alpha1.PersistentVolume{
		Metadata: convertObjectMeta(persistentVolume.ObjectMeta),
		Spec:     spec,
		Status:   convertPersistentVolumeStatus(*persistentVolume),
	}
}

// func convertProto2PersistentVolumeClaim(req *corev1alpha1.PersistentVolume) *corev1.PersistentVolumeClaim {
// 	accessModes := make([]corev1.PersistentVolumeAccessMode, 0)
// 	for _, mode := range req.GetSpec().GetAccessModes() {
// 		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode.String()))
// 	}
// 	quantity, err := resource.ParseQuantity(req.GetSpec().GetCapacity())
// 	if err != nil {
// 		return nil
// 	}
// 	return &corev1.PersistentVolumeClaim{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:        req.GetMetadata().GetName(),
// 			Namespace:   req.GetMetadata().GetNamespace(),
// 			Annotations: req.GetMetadata().GetAnnotations(),
// 			Labels:      req.GetMetadata().GetLabels(),
// 		},
// 		Spec: corev1.PersistentVolumeClaimSpec{
// 			AccessModes: accessModes,
// 			Resources: corev1.VolumeResourceRequirements{
// 				Requests: corev1.ResourceList{
// 					corev1.ResourceStorage: quantity,
// 				},
// 			},
// 			StorageClassName: func() *string { sc := req.GetSpec().GetStorageClassName(); return &sc }(),
// 		},
// 	}
// }

func convertPersistentVolumeClaims2Proto(pvcs []*corev1.PersistentVolumeClaim) []*corev1alpha1.PersistentVolumeClaim {
	res := make([]*corev1alpha1.PersistentVolumeClaim, 0, len(pvcs))
	for _, pvc := range pvcs {
		res = append(res, convertPersistentVolumeClaim2Proto(pvc))
	}
	return res
}

func convertPersistentVolumeClaim2Proto(req *corev1.PersistentVolumeClaim) *corev1alpha1.PersistentVolumeClaim {
	if req == nil {
		klog.ErrorS(nil, "Received nil PersistentVolumeClaim input for conversion", "function", "convertPersistentVolumeClaim2Proto")
		return nil
	}
	accessModes := make([]corev1alpha1.PersistentVolumeAccessMode, 0)
	for _, mode := range req.Spec.AccessModes {
		accessModes = append(accessModes, corev1alpha1.PersistentVolumeAccessMode(corev1alpha1.PersistentVolumeAccessMode_value[string(mode)]))
	}
	if req.Spec.StorageClassName == nil {
		defaultName := "cluster-default"
		req.Spec.StorageClassName = &defaultName
	}
	return &corev1alpha1.PersistentVolumeClaim{
		Metadata: convertObjectMeta(req.ObjectMeta),
		Spec: &corev1alpha1.PersistentVolumeClaimSpec{
			AccessModes:      accessModes,
			Resources:        &corev1alpha1.ResourceRequirements{Requests: &corev1alpha1.ResourceList{Storage: req.Spec.Resources.Requests.Storage().String()}},
			StorageClassName: *req.Spec.StorageClassName,
		},
		Status: &corev1alpha1.PersistentVolumeClaimStatus{
			Phase:         convertPVCPhase(req.Status.Phase),
			AccessModes:   convertAccessModes(req.Status.AccessModes),
			Capacity:      convertStorageResourceList(req.Status.Capacity),
			Conditions:    convertPersistentVolumeClaimStatusConditions(req.Status.Conditions),
			SnapshotCount: 0,
			PodName:       nil,
		},
	}
}

func convertPersistentVolumeClaimStatusConditions(conditions []corev1.PersistentVolumeClaimCondition) []*corev1alpha1.PersistentVolumeClaimCondition {
	result := make([]*corev1alpha1.PersistentVolumeClaimCondition, 0, len(conditions))
	for _, condition := range conditions {
		result = append(result, &corev1alpha1.PersistentVolumeClaimCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastProbeTime:      condition.LastProbeTime.Unix(),
			LastTransitionTime: condition.LastTransitionTime.Unix(),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}
	return result
}

func convertStorageResourceList(resourceList corev1.ResourceList) *corev1alpha1.ResourceList {
	if resourceList == nil {
		return &corev1alpha1.ResourceList{}
	}
	return &corev1alpha1.ResourceList{
		Storage: filterResource(resourceList.Storage().String()),
	}
}

func filterResource(resource string) string {
	if resource == "0" {
		return ""
	}
	return resource
}

func convertAccessModes(modes []corev1.PersistentVolumeAccessMode) []corev1alpha1.PersistentVolumeAccessMode {
	result := make([]corev1alpha1.PersistentVolumeAccessMode, 0, len(modes))
	for _, mode := range modes {
		result = append(result, corev1alpha1.PersistentVolumeAccessMode(corev1alpha1.PersistentVolumeAccessMode_value[string(mode)]))
	}
	return result
}

func convertPVCPhase(phase corev1.PersistentVolumeClaimPhase) corev1alpha1.PVCPhase {
	switch phase {
	case corev1.ClaimPending:
		return corev1alpha1.PVCPhase_PVC_Pending
	case corev1.ClaimBound:
		return corev1alpha1.PVCPhase_PVC_Bound
	case corev1.ClaimLost:
		return corev1alpha1.PVCPhase_PVC_Lost
	}
	panic("unexpected PVC phase")
}
