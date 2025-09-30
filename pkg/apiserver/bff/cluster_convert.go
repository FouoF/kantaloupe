package bff

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clustersv1alpha1 "github.com/dynamia-ai/kantaloupe/api/clusters/v1alpha1"
	clustercrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/api/types"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/helper"
)

// ConvertClusters2Proto converts clusters cr to protobuf clusters.
func ConvertClusters2Proto(clusters []*clustercrdv1alpha1.Cluster, metrics map[string]clusterMetric) []*clustersv1alpha1.Cluster {
	res := make([]*clustersv1alpha1.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		metric, ok := metrics[cluster.GetName()]
		if ok {
			res = append(res, ConvertCluster2Proto(cluster, &metric))
		} else {
			res = append(res, ConvertCluster2Proto(cluster, nil))
		}
	}

	return res
}

// ConvertCluster2Proto converts cluster cr to protobuf cluster.
func ConvertCluster2Proto(cluster *clustercrdv1alpha1.Cluster, metric *clusterMetric) *clustersv1alpha1.Cluster {
	spec := &clustersv1alpha1.ClusterSpec{
		Provider:          clustersv1alpha1.ClusterProvider(clustersv1alpha1.ClusterProvider_value[cluster.Spec.Provider]),
		Type:              clustersv1alpha1.ClusterType(clustersv1alpha1.ClusterType_value[cluster.Spec.Type]),
		ApiEndpoint:       cluster.Spec.APIEndpoint,
		AliasName:         getAliasNameFrom(cluster.Annotations),
		Description:       cluster.Annotations[constants.ClusterDescriptionAnnotationKey],
		PrometheusAddress: cluster.Spec.PrometheusAddress,
		GatewayAddress:    cluster.Spec.GatewayAddress,
	}

	status := &clustersv1alpha1.ClusterStatus{
		KubernetesVersion:     cluster.Status.KubernetesVersion,
		KubeSystemID:          cluster.Status.KubeSystemID,
		NodeSummary:           convertResourceSummary2Proto(cluster.Status.NodeSummary),
		PodSummary:            convertResourceSummary2Proto(cluster.Status.PodSetSummary),
		KantaloupeflowSummary: convertResourceSummary2Proto(cluster.Status.KantaloupeflowSummary),
		Conditions:            convertCondition2Proto(cluster.Status.Conditions),
		State:                 convertCondition2State(cluster.Status),
	}

	if metric != nil {
		status.CpuUsage = metric.cpuUsage
		status.MemoryUsage = metric.memoryUsage
		status.GpuCoreUsage = metric.gpuCoreUsage
		status.GpuMemoryUsage = metric.gpuMemoryUsage
		status.GpuCoreAllocated = metric.gpuCoreAllocated
		status.GpuMemoryAllocated = metric.gpuMemoryAllocated
	}

	// cpu_total, mem_total, gpu_total, KantaloupeflowSummary,
	if cluster.Status.ResourceSummary != nil {
		if val, ok := cluster.Status.ResourceSummary.Allocatable["nvidia.com/gpu.count"]; ok {
			status.GpuTotal = int32(val.Value())
		}
		if val, ok := cluster.Status.ResourceSummary.Allocatable["nvidia.com/gpu-memory.count"]; ok {
			status.GpuMemoryTotal = val.Value()
		}
		if val, ok := cluster.Status.ResourceSummary.Allocatable[corev1.ResourceCPU]; ok {
			status.CpuTotal = int32(val.Value())
		}
		if val, ok := cluster.Status.ResourceSummary.Allocatable[corev1.ResourceMemory]; ok {
			status.MemoryTotal = val.Value()
		}
	}

	return &clustersv1alpha1.Cluster{
		Metadata: convertObjectMeta(cluster.ObjectMeta),
		Spec:     spec,
		Status:   status,
	}
}

func convertResourceSummary2Proto(nodes *clustercrdv1alpha1.ResourceSummary) *clustersv1alpha1.ResourceSummary {
	totalNum, readyNum := int32(-1), int32(-1)
	if nodes != nil {
		totalNum = nodes.TotalNum
		readyNum = nodes.ReadyNum
	}

	return &clustersv1alpha1.ResourceSummary{
		TotalNum: totalNum,
		ReadyNum: readyNum,
	}
}

func convertCondition2Proto(condition []metav1.Condition) []*types.Condition {
	res := []*types.Condition{}
	for _, c := range condition {
		res = append(res, &types.Condition{
			Message:            c.Message,
			Reason:             c.Reason,
			Status:             string(c.Status),
			Type:               c.Type,
			LastTransitionTime: c.LastTransitionTime.Time.Format("2006-01-02 15:04:05"),
		})
	}
	return res
}

func getAliasNameFrom(annotations map[string]string) string {
	return annotations[constants.ClusterAliasAnnotationKey]
}

func convertCondition2State(status clustercrdv1alpha1.ClusterStatus) clustersv1alpha1.ClusterState {
	if helper.IsClusterReady(&status) {
		return clustersv1alpha1.ClusterState_RUNNING
	}
	return clustersv1alpha1.ClusterState_UNHEALTH
}
