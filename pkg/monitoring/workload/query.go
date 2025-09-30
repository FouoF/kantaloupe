package workload

import (
	"fmt"
	"sort"

	monitoringv1alpha1 "github.com/dynamia-ai/kantaloupe/api/monitoring/v1alpha1"
)

// TODO: Replace Pod with custom workload type when available.
const (
	maxCategories  = 5
	maxGPUsPerNode = 8
)

// NodeGPUWorkloads represents workloads on each GPU of a node.
type NodeGPUWorkloads struct {
	GPUWorkloads map[string]int32
}

// ClusterNodeWorkloads represents workloads on each node in the cluster.
type ClusterNodeWorkloads struct {
	NodeWorkloads map[string]int32
}

// GetNodeDistribution returns the workload distribution for a specific node's GPUs.
func GetNodeDistribution(workloads NodeGPUWorkloads) []*monitoringv1alpha1.DistributionPoint {
	points := make([]*monitoringv1alpha1.DistributionPoint, 0, len(workloads.GPUWorkloads))

	// Sort GPUs by UUID for consistent ordering
	gpuUUIDs := make([]string, 0, len(workloads.GPUWorkloads))
	for uuid := range workloads.GPUWorkloads {
		gpuUUIDs = append(gpuUUIDs, uuid)
	}
	sort.Strings(gpuUUIDs)

	// Create distribution points
	for _, uuid := range gpuUUIDs {
		points = append(points, &monitoringv1alpha1.DistributionPoint{
			Name:  uuid,
			Value: workloads.GPUWorkloads[uuid],
		})
	}

	return points
}

// GetClusterDistribution returns the workload distribution across the cluster.
func GetClusterDistribution(workloads ClusterNodeWorkloads) []*monitoringv1alpha1.DistributionPoint {
	if len(workloads.NodeWorkloads) == 0 {
		return nil
	}

	var minVal, maxVal int32 = -1, 0
	for _, count := range workloads.NodeWorkloads {
		if minVal == -1 || count < minVal {
			minVal = count
		}
		if count > maxVal {
			maxVal = count
		}
	}

	// Calculate range size and create buckets
	rangeSize := int32(10)

	// Initialize buckets
	type bucket struct {
		start int32
		end   int32
		count int32
	}
	buckets := make([]bucket, 0, maxCategories)

	// Create bucket ranges
	for start := int32(0); start <= maxVal; start += rangeSize {
		end := start + rangeSize - 1
		buckets = append(buckets, bucket{start: start, end: end, count: 0})
		if end == maxVal {
			break
		}
	}

	if maxVal > 40 {
		buckets[maxCategories-1].end = 999
	}

	// Count nodes in each bucket
	for _, count := range workloads.NodeWorkloads {
		for i := range buckets {
			if count >= buckets[i].start && count <= buckets[i].end {
				buckets[i].count++
				break
			}
		}
	}

	// Convert buckets to distribution points
	points := make([]*monitoringv1alpha1.DistributionPoint, 0, len(buckets))
	for _, b := range buckets {
		if b.count > 0 { // Only include non-empty buckets
			var name string
			if b.start == b.end {
				name = fmt.Sprintf("%d", b.start)
			} else {
				name = fmt.Sprintf("%d-%d", b.start, b.end)
			}
			points = append(points, &monitoringv1alpha1.DistributionPoint{
				Name:  name,
				Value: b.count,
			})
		}
	}

	return points
}

// GetNodeWorkloadQueries returns the Prometheus queries for node workload distribution.
func GetNodeWorkloadQueries(cluster, node string) string {
	// TODO: Replace with actual workload query when custom workload type is available
	return fmt.Sprintf(`count(vGPUPodsDeviceAllocated{cluster="%s", nodename="%s"}) by (deviceuuid)`, cluster, node)
}
