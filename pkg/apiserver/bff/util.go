package bff

import (
	monitoringv1alpha1 "github.com/dynamia-ai/kantaloupe/api/monitoring/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/api/types"
	"github.com/dynamia-ai/kantaloupe/pkg/service/monitoring"
)

const (
	NvidiaGpuCount = "nvidia.com/gpu.count"
	MetaxGpuCount  = "metax-tech.com/gpu-device.mode"
)

func NewEmptyPage(page, size int32) *types.Pagination {
	return &types.Pagination{
		Page:     page,
		PageSize: size,
		Total:    0,
		Pages:    0,
	}
}

func ConvertResourceType2QueryType(resourceType monitoringv1alpha1.ResourceType) map[string]monitoring.QueryType {
	switch resourceType {
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_CPU:
		return map[string]monitoring.QueryType{
			"total":     monitoring.QueryTypeCPUTotal,
			"used":      monitoring.QueryTypeCPUUsed,
			"allocated": monitoring.QueryTypeCPUAllocated,
		}
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_MEMORY:
		return map[string]monitoring.QueryType{
			"total":     monitoring.QueryTypeMemoryTotal,
			"used":      monitoring.QueryTypeMemoryUsed,
			"allocated": monitoring.QueryTypeMemoryAllocated,
		}
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_CORE:
		return map[string]monitoring.QueryType{
			"total":     monitoring.QueryTypeGPUCoreTotal,
			"used":      monitoring.QueryTypeGPUCoreUsed,
			"allocated": monitoring.QueryTypeGPUCoreAllocated,
		}
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_MEMORY:
		return map[string]monitoring.QueryType{
			"total":     monitoring.QueryTypeGPUMemoryTotal,
			"used":      monitoring.QueryTypeGPUMemoryUsed,
			"allocated": monitoring.QueryTypeGPUMemoryAllocated,
		}
	}
	return map[string]monitoring.QueryType{}
}

func ConvertResourceType2GPUQueryType(resourceType monitoringv1alpha1.ResourceType) map[string]monitoring.GPUQueryType {
	switch resourceType {
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_CORE:
		return map[string]monitoring.GPUQueryType{
			"total":     monitoring.GPUQueryTypeCoreTotal,
			"used":      monitoring.GPUQueryTypeCoreUsed,
			"allocated": monitoring.GPUQueryTypeCoreAllocated,
		}
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_MEMORY:
		return map[string]monitoring.GPUQueryType{
			"total":     monitoring.GPUQueryTypeMemoryTotal,
			"used":      monitoring.GPUQueryTypeMemoryUsed,
			"allocated": monitoring.GPUQueryTypeMemoryAllocated,
		}
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_TEMP:
		return map[string]monitoring.GPUQueryType{
			"used": monitoring.GPUQueryTypeTemperature,
		}
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_POWER:
		return map[string]monitoring.GPUQueryType{
			"used": monitoring.GPUQueryTypePower,
		}
	}
	return map[string]monitoring.GPUQueryType{}
}
