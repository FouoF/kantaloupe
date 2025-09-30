package bff

import (
	"context"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/klog/v2"

	corev1alpha1 "github.com/dynamia-ai/kantaloupe/api/core/v1alpha1"
	storagev1alpha1 "github.com/dynamia-ai/kantaloupe/api/storage/v1alpha1"
	kantaloupeapi "github.com/dynamia-ai/kantaloupe/api/v1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/service/core"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
)

type StorageHandler struct {
	kantaloupeapi.UnimplementedStorageServer
	workloadService core.Service
}

func NewStorageHandler(clientManager engine.ClientManagerInterface) *StorageHandler {
	return &StorageHandler{
		workloadService: core.NewService(clientManager),
	}
}

func (h *StorageHandler) ListStorageClasses(ctx context.Context, req *storagev1alpha1.ListStorageClassesRequest) (*storagev1alpha1.ListStorageClassesResponse, error) {
	storageClasses, err := h.workloadService.ListStorageClasses(ctx, req.GetCluster())
	if err != nil {
		return nil, err
	}
	items := convertStorageClasses(storageClasses)
	if err = utils.SortStructSlice(items, req.GetSortOption().GetField(), req.GetSortOption().GetAsc(), utils.SnakeToCamelMapper()); err != nil {
		return nil, err
	}

	// TODO: why not page?
	return &storagev1alpha1.ListStorageClassesResponse{
		Items:      items,
		Pagination: utils.NewPage(req.Page, req.PageSize, len(storageClasses)),
	}, nil
}

func (h *StorageHandler) CreateStorage(ctx context.Context, req *storagev1alpha1.CreateStorageRequest) (*storagev1alpha1.Storage, error) {
	if req.GetStorageName() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "storage name can not be empty")
	}
	if req.GetNamespace() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "namespace can not be empty")
	}
	if req.GetStorageSize() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "storage size can not be empty")
	}
	if req.GetAccessMode() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "access mode can not be empty")
	}
	if errs := validation.IsDNS1035Label(req.GetCluster()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cluster name %s is invalid, error: %s", req.GetCluster(), errs)
	}
	if req.GetStorageType() == storagev1alpha1.StorageType_PVC {
		// dynamic PVC must have a StorageClassName
		if req.StorageClassName == "" {
			return nil, status.Errorf(codes.InvalidArgument, "storageClass name is required for StorageType_PVC")
		}
	}

	var pvc *corev1.PersistentVolumeClaim

	switch req.GetStorageType() {
	case storagev1alpha1.StorageType_LocalPV:
		klog.V(4).InfoS("Handling LocalPV storage creation", "storageName", req.GetStorageName(), "namespace", req.GetNamespace())

		localPV, err := h.convertRequestToLocalPV(req)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to build Local PersistentVolume: %v", err)
		}

		createdPV, err := h.workloadService.CreatePersistentVolume(ctx, req.GetCluster(), localPV)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create Local PersistentVolume: %v", err)
		}

		localPVC, err := h.convertRequestToPVC(req, createdPV.Spec.StorageClassName)
		if err != nil {
			_ = h.workloadService.DeletePersistentVolume(ctx, req.GetCluster(), createdPV.Name) // try to roll back
			return nil, status.Errorf(codes.Internal, "failed to build PersistentVolumeClaim for LocalPV: %v", err)
		}

		localPVC.Spec.VolumeName = createdPV.Name
		pvc, err = h.workloadService.CreatePersistentVolumeClaim(ctx, req.GetCluster(), req.GetNamespace(), localPVC, req.GetStorageType())
		if err != nil {
			_ = h.workloadService.DeletePersistentVolume(ctx, req.GetCluster(), createdPV.Name) // try to roll back
			return nil, status.Errorf(codes.Internal, "failed to create PersistentVolumeClaim for LocalPV: %v", err)
		}
	case storagev1alpha1.StorageType_NFS:
		nfsPV, err := h.convertRequestToNFSPV(req)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to build NFS PersistentVolume: %v", err)
		}
		createdPV, err := h.workloadService.CreatePersistentVolume(ctx, req.GetCluster(), nfsPV)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create NFS PersistentVolume: %v", err)
		}
		nfsPVC, err := h.convertRequestToPVC(req, createdPV.Spec.StorageClassName)
		if err != nil {
			_ = h.workloadService.DeletePersistentVolume(ctx, req.GetCluster(), createdPV.Name) // try to roll back
			return nil, status.Errorf(codes.Internal, "failed to build PersistentVolumeClaim for NFS: %v", err)
		}
		nfsPVC.Spec.VolumeName = createdPV.Name
		pvc, err = h.workloadService.CreatePersistentVolumeClaim(ctx, req.GetCluster(), req.GetNamespace(), nfsPVC, req.GetStorageType())
		if err != nil {
			_ = h.workloadService.DeletePersistentVolume(ctx, req.GetCluster(), createdPV.Name) // try to roll back
			return nil, status.Errorf(codes.Internal, "failed to create PersistentVolumeClaim for NFS: %v", err)
		}
	case storagev1alpha1.StorageType_PVC: // dynamic PVC
		dynamicPVC, err := h.convertRequestToPVC(req, "")
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to build PersistentVolumeClaim for dynamic PVC: %v", err)
		}
		pvc, err = h.workloadService.CreatePersistentVolumeClaim(ctx, req.GetCluster(), req.GetNamespace(), dynamicPVC, req.GetStorageType())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create PersistentVolumeClaim for dynamic PVC: %v", err)
		}
	case storagev1alpha1.StorageType_StorageTypeUnspecified:
		return nil, status.Errorf(codes.InvalidArgument, "storageType is required and cannot be unspecified")
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported storage type: %s", req.GetStorageType().String())
	}

	response := &storagev1alpha1.Storage{
		PersistentVolumeClaim: convertPersistentVolumeClaim2Proto(pvc),
		StorageType:           req.GetStorageType(),
	}
	return response, nil
}

func (h *StorageHandler) DeleteStorage(ctx context.Context, req *storagev1alpha1.DeleteStorageRequest) (*emptypb.Empty, error) {
	err := h.workloadService.DeletePersistentVolumeClaim(ctx, req.GetCluster(), req.GetNamespace(), req.GetName())
	if err != nil {
		klog.V(4).ErrorS(err, "failed to delete PersistentVolumeClaim")
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (h *StorageHandler) ListStorages(ctx context.Context, req *storagev1alpha1.ListStoragesRequest) (*storagev1alpha1.ListStoragesResponse, error) {
	pvcs, err := h.workloadService.ListPersistentVolumeClaims(ctx, req.GetCluster(), req.GetNamespace(), req.GetIsManage())
	if err != nil {
		return nil, err
	}

	filtered := filterStorages(pvcs, req.GetName(), req.GetPhase(), req.GetStorageType())

	// TODO: use utils.SortStructSlice
	sortDir := constants.SortByDesc
	if req.GetSortOption().GetAsc() {
		sortDir = "asc"
	}

	sortPvcByMetaFields(filtered, req.GetSortOption().GetField(), sortDir)
	paged := utils.PagedItems(filtered, req.GetPage(), req.GetPageSize())
	items := convertPersistentVolumeClaims2Proto(paged)

	return &storagev1alpha1.ListStoragesResponse{
		Items:      items,
		Pagination: utils.NewPage(req.GetPage(), req.GetPageSize(), len(filtered)),
	}, nil
}

func sortPvcByMetaFields(list []*corev1.PersistentVolumeClaim, field, asc string) {
	sort.Slice(list, func(i, j int) bool {
		var less bool

		switch field {
		case "field_name":
			less = list[i].GetName() < list[j].GetName()
		case "created_at":
			less = !list[i].GetCreationTimestamp().After(list[j].GetCreationTimestamp().Time)
		default:
			less = !list[i].GetCreationTimestamp().After(list[j].GetCreationTimestamp().Time)
		}

		if asc == constants.SortByDesc {
			return !less
		}
		return less
	})
}

func filterStorages(pvcs []*corev1.PersistentVolumeClaim, keyword string, phase corev1alpha1.PVCPhase, storageType storagev1alpha1.StorageType) []*corev1.PersistentVolumeClaim {
	res := []*corev1.PersistentVolumeClaim{}
	for _, pvc := range pvcs {
		if !utils.MatchByFuzzyName(pvc, keyword) {
			continue
		}

		pvcPhase := convertPVCPhase(pvc.Status.Phase)
		if phase != corev1alpha1.PVCPhase_PVC_PHASE_UNSPECIFIED && pvcPhase != phase {
			continue
		}

		// filter by storage type
		storageTypeStr := pvc.Annotations[constants.StorageTypeKey]
		st := storagev1alpha1.StorageType(storagev1alpha1.StorageType_value[storageTypeStr])
		if storageType != storagev1alpha1.StorageType_StorageTypeUnspecified && st != storageType {
			continue
		}
		res = append(res, pvc)
	}
	return res
}
