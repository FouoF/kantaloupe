package bff

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	quotav1alpha1 "github.com/dynamia-ai/kantaloupe/api/quotas/v1alpha1"
	kantaloupeapi "github.com/dynamia-ai/kantaloupe/api/v1"
	kfservice "github.com/dynamia-ai/kantaloupe/pkg/service/kantaloupeflow"
	quotaservice "github.com/dynamia-ai/kantaloupe/pkg/service/quota"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
)

var _ kantaloupeapi.QuotaServer = &QuotaHandler{}

// QuotaHandler handles resource quota related API requests.
type QuotaHandler struct {
	quotaService    quotaservice.Service
	workloadService kfservice.Service
	kantaloupeapi.UnimplementedQuotaServer
}

// NewQuotaHandler creates a new quota handler.
func NewQuotaHandler(quotaService quotaservice.Service, workloadService kfservice.Service) *QuotaHandler {
	return &QuotaHandler{
		quotaService:    quotaService,
		workloadService: workloadService,
	}
}

// ListQuotas lists resource quotas with optional filtering.
func (h *QuotaHandler) ListQuotas(ctx context.Context, req *quotav1alpha1.ListQuotasRequest) (*quotav1alpha1.ListQuotasResponse, error) {
	// Get quotas from service
	quotas, err := h.quotaService.ListQuotas(ctx, req.GetNamespace(), req.GetCluster())
	if err != nil {
		klog.ErrorS(err, "failed to list resource quotas", "Error:", err)
		return nil, status.Errorf(codes.Internal, "failed to list resource quotas: %v", err)
	}

	filtered := filterQuotas(quotas, req.GetName())

	// TODO: "createdTime" -> "metadata.creationTimestamp"
	if req.SortOption.GetField() == "createdTime" {
		req.SortOption.Field = "creationTimestamp"
	}

	if err := utils.SortStructSlice(filtered, req.SortOption.GetField(), req.SortOption.GetAsc(), utils.SnakeToCamelMapper()); err != nil {
		return nil, err
	}

	paged := utils.PagedItems(filtered, req.GetPage(), req.GetPageSize())

	// Convert to response objects
	items := convertQuotasToResponses(paged)
	for i := range items {
		workloads, err := h.workloadService.ListKantaloupeflows(ctx, req.GetCluster(), items[i].Namespace)
		if err != nil {
			klog.ErrorS(err, "failed to list kantaloupeflows", "Error:", err)
			return nil, status.Errorf(codes.Internal, "failed to list kantaloupeflows: %v", err)
		}
		for _, workload := range workloads {
			items[i].Workload = append(items[i].Workload, workload.Name)
		}
	}

	return &quotav1alpha1.ListQuotasResponse{
		Items:      items,
		Pagination: utils.NewPage(req.Page, req.PageSize, len(filtered)),
	}, nil
}

func filterQuotas(quotas []*corev1.ResourceQuota, keyword string) []*corev1.ResourceQuota {
	gpuKeys := []string{"requests.nvidia.com/gpucores", "requests.nvidia.com/gpumem", "limits.nvidia.com/gpucores", "limits.nvidia.com/gpumem"}
	res := []*corev1.ResourceQuota{}
	for _, quota := range quotas {
		if !utils.MatchByFuzzyName(quota, keyword) {
			continue
		}

		for _, key := range gpuKeys {
			if _, ok := quota.Spec.Hard[corev1.ResourceName(key)]; ok {
				res = append(res, quota)
				break
			}
		}
	}
	return res
}

// CreateQuota creates a new resource quota.
func (h *QuotaHandler) CreateQuota(ctx context.Context, req *quotav1alpha1.CreateQuotaRequest) (*quotav1alpha1.QuotaResponse, error) {
	// Validate request
	if err := validateCreateQuotaRequest(req); err != nil {
		return nil, err
	}

	// Create the resource quota
	quota, err := h.quotaService.CreateQuota(
		ctx,
		req.GetName(),
		req.GetNamespace(),
		req.GetCluster(),
		req.GetHard(),
	)
	if err != nil {
		klog.ErrorS(err, "failed to create resource quota", "name", req.GetName(), "Error:", err)
		return nil, status.Errorf(codes.Internal, "failed to create resource quota: %v", err)
	}

	// Convert to response
	return convertQuotaToResponse(quota)
}

func (h *QuotaHandler) GetQuota(ctx context.Context, req *quotav1alpha1.GetQuotaRequest) (*quotav1alpha1.QuotaResponse, error) {
	quota, err := h.quotaService.GetQuota(ctx, req.GetName(), req.GetNamespace(), req.GetCluster())
	if err != nil {
		klog.ErrorS(err, "failed to get resource quota", "name", req.GetName(), "Error:", err)
		return nil, status.Errorf(codes.Internal, "failed to get resource quota: %v", err)
	}
	return convertQuotaToResponse(quota)
}

// UpdateQuota updates an existing resource quota.
func (h *QuotaHandler) UpdateQuota(ctx context.Context, req *quotav1alpha1.UpdateQuotaRequest) (*quotav1alpha1.QuotaResponse, error) {
	// Validate request
	if err := validateUpdateQuotaRequest(req); err != nil {
		return nil, err
	}

	// Update the resource quota
	quota, err := h.quotaService.UpdateQuota(
		ctx,
		req.GetName(),
		req.GetNamespace(),
		req.GetCluster(),
		req.GetHard(),
	)
	if err != nil {
		klog.ErrorS(err, "failed to update resource quota", "name", req.GetName(), "Error:", err)
		return nil, status.Errorf(codes.Internal, "failed to update resource quota: %v", err)
	}

	// Convert to response
	return convertQuotaToResponse(quota)
}

// DeleteQuota deletes a resource quota.
func (h *QuotaHandler) DeleteQuota(ctx context.Context, req *quotav1alpha1.DeleteQuotaRequest) (*emptypb.Empty, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "quota name cannot be empty")
	}

	// Delete quota
	err := h.quotaService.DeleteQuota(ctx, req.GetName(), req.GetNamespace(), req.GetCluster())
	if err != nil {
		klog.ErrorS(err, "failed to delete resource quota", "name", req.GetName(), "Error:", err)
		return nil, status.Errorf(codes.Internal, "failed to delete resource quota: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// validateCreateQuotaRequest validates CreateQuotaRequest.
func validateCreateQuotaRequest(req *quotav1alpha1.CreateQuotaRequest) error {
	if req.GetName() == "" {
		return status.Error(codes.InvalidArgument, "quota name cannot be empty")
	}

	if len(req.GetHard()) == 0 {
		return status.Error(codes.InvalidArgument, "quota hard limits cannot be empty")
	}

	return nil
}

// validateUpdateQuotaRequest validates UpdateQuotaRequest.
func validateUpdateQuotaRequest(req *quotav1alpha1.UpdateQuotaRequest) error {
	if req.GetName() == "" {
		return status.Error(codes.InvalidArgument, "quota name cannot be empty")
	}

	if len(req.GetHard()) == 0 {
		return status.Error(codes.InvalidArgument, "quota hard limits cannot be empty")
	}

	return nil
}
