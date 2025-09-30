package resource

import (
	"fmt"

	monitoringv1alpha1 "github.com/dynamia-ai/kantaloupe/api/monitoring/v1alpha1"
)

// Queries contains the allocated and used queries for a resource.
type Queries struct {
	Allocated string
	Used      string
}

// GetTopNodesQuery returns a query for getting top K nodes by resource usage.
func GetTopNodesQuery(resourceType monitoringv1alpha1.ResourceType, rankingType monitoringv1alpha1.RankingType, cluster string, limit int32) (string, error) {
	var query string
	switch resourceType {
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_UNSPECIFIED:
		return "", fmt.Errorf("unspecified resource type")
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_CPU:
		if rankingType == monitoringv1alpha1.RankingType_RANKING_TYPE_ALLOCATED {
			query = `topk(%d, kantaloupe_node_cpu_allocated{cluster="%s"} / kantaloupe_node_cpu_total{cluster="%s"} * 100)`
		} else {
			query = `topk(%d, kantaloupe_node_cpu_used{cluster="%s"} / kantaloupe_node_cpu_total{cluster="%s"} * 100)`
		}
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_MEMORY:
		if rankingType == monitoringv1alpha1.RankingType_RANKING_TYPE_ALLOCATED {
			query = `topk(%d, kantaloupe_node_mem_allocated{cluster="%s"} / kantaloupe_node_mem_total{cluster="%s"} * 100)`
		} else {
			query = `topk(%d, kantaloupe_node_mem_used{cluster="%s"} / kantaloupe_node_mem_total{cluster="%s"} * 100)`
		}
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_CORE:
		if rankingType == monitoringv1alpha1.RankingType_RANKING_TYPE_ALLOCATED {
			query = `topk(%d, kantaloupe_node_gpucore_allocated{cluster="%s"} / on(node) kantaloupe_node_gpucore_total{cluster="%s"} * 100)`
		} else {
			query = `topk(%d, kantaloupe_node_gpucore_used{cluster="%s"} / on(node) kantaloupe_node_gpucore_total{cluster="%s"} * 100)`
		}
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_MEMORY:
		if rankingType == monitoringv1alpha1.RankingType_RANKING_TYPE_ALLOCATED {
			query = `topk(%d, kantaloupe_node_gpumem_allocated{cluster="%s"} / on(node) kantaloupe_node_gpumem_total{cluster="%s"} * 100)`
		} else {
			query = `topk(%d, kantaloupe_node_gpumem_used{cluster="%s"} / on(node) kantaloupe_node_gpu_total{cluster="%s"} * 100)`
		}
	default:
		return "", fmt.Errorf("unsupported resource type: %v", resourceType)
	}
	return fmt.Sprintf(query, limit, cluster, cluster), nil
}
