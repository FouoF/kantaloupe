package bff

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	quotav1alpha1 "github.com/dynamia-ai/kantaloupe/api/quotas/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
)

// convertQuotasToResponses converts Kubernetes ResourceQuota objects to quota responses.
func convertQuotasToResponses(quotas []*corev1.ResourceQuota) []*quotav1alpha1.QuotaResponse {
	items := make([]*quotav1alpha1.QuotaResponse, 0, len(quotas))
	for _, quota := range quotas {
		resp, err := convertQuotaToResponse(quota)
		if err != nil {
			klog.ErrorS(err, "Failed to convert quota")
			continue
		}
		items = append(items, resp)
	}
	return items
}

// convertQuotaToResponse converts a Kubernetes ResourceQuota to a quota response.
func convertQuotaToResponse(quota *corev1.ResourceQuota) (*quotav1alpha1.QuotaResponse, error) {
	if quota == nil {
		return nil, status.Error(codes.Internal, "quota is nil")
	}
	isManaged := false
	if v, ok := quota.Labels[constants.ManagedByLabelKey]; ok && v == constants.ManagedByLabelValue {
		isManaged = true
	}

	// Convert hard limits to string map
	hard := make(map[string]string)
	for k, v := range quota.Spec.Hard {
		hard[string(k)] = v.String()
	}

	// Convert used amounts to string map
	used := make(map[string]string)
	for k, v := range quota.Status.Used {
		used[string(k)] = v.String()
	}

	return &quotav1alpha1.QuotaResponse{
		Name:        quota.Name,
		Namespace:   quota.Namespace,
		CreatedTime: quota.CreationTimestamp.Unix(),
		Hard:        hard,
		Used:        used,
		Labels:      quota.Labels,
		IsManaged:   isManaged,
	}, nil
}
