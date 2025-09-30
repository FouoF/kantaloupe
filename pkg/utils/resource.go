package utils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/resourcehelper"
)

// Resource is a collection of compute resource.
type Resource struct {
	MilliCPU         int64
	Memory           int64
	EphemeralStorage int64
	AllowedPodNumber int64

	// ScalarResources
	ScalarResources map[corev1.ResourceName]int64
}

// EmptyResource creates a empty resource object and returns.
func EmptyResource() *Resource {
	return &Resource{}
}

// NewResource creates a new resource object from resource list.
func NewResource(rl corev1.ResourceList) *Resource {
	r := &Resource{}
	for rName, rQuant := range rl {
		switch rName {
		case corev1.ResourceCPU:
			r.MilliCPU += rQuant.MilliValue()
		case corev1.ResourceMemory:
			r.Memory += rQuant.Value()
		case corev1.ResourcePods:
			r.AllowedPodNumber += rQuant.Value()
		case corev1.ResourceEphemeralStorage:
			r.EphemeralStorage += rQuant.Value()
		default:
			if resourcehelper.IsScalarResourceName(rName) {
				r.AddScalar(rName, rQuant.Value())
			}
		}
	}
	return r
}

// Add is used to add two resources.
func (r *Resource) Add(rl corev1.ResourceList) {
	if r == nil {
		return
	}

	for rName, rQuant := range rl {
		switch rName {
		case corev1.ResourceCPU:
			r.MilliCPU += rQuant.MilliValue()
		case corev1.ResourceMemory:
			r.Memory += rQuant.Value()
		case corev1.ResourcePods:
			r.AllowedPodNumber += rQuant.Value()
		case corev1.ResourceEphemeralStorage:
			r.EphemeralStorage += rQuant.Value()
		default:
			if resourcehelper.IsScalarResourceName(rName) {
				r.AddScalar(rName, rQuant.Value())
			}
		}
	}
}

// SetMaxResource compares with ResourceList and takes max value for each Resource.
func (r *Resource) SetMaxResource(rl corev1.ResourceList) {
	if r == nil {
		return
	}

	for rName, rQuant := range rl {
		switch rName {
		case corev1.ResourceCPU:
			cpu := rQuant.MilliValue()
			if cpu > r.MilliCPU {
				r.MilliCPU = cpu
			}
		case corev1.ResourceMemory:
			if mem := rQuant.Value(); mem > r.Memory {
				r.Memory = mem
			}
		case corev1.ResourceEphemeralStorage:
			if ephemeralStorage := rQuant.Value(); ephemeralStorage > r.EphemeralStorage {
				r.EphemeralStorage = ephemeralStorage
			}
		case corev1.ResourcePods:
			if pods := rQuant.Value(); pods > r.AllowedPodNumber {
				r.AllowedPodNumber = pods
			}
		default:
			if resourcehelper.IsScalarResourceName(rName) {
				if value := rQuant.Value(); value > r.ScalarResources[rName] {
					r.SetScalar(rName, value)
				}
			}
		}
	}
}

// AddScalar adds a resource by a scalar value of this resource.
func (r *Resource) AddScalar(name corev1.ResourceName, quantity int64) {
	r.SetScalar(name, r.ScalarResources[name]+quantity)
}

// SetScalar sets a resource by a scalar value of this resource.
func (r *Resource) SetScalar(name corev1.ResourceName, quantity int64) {
	// Lazily allocate scalar resource map.
	if r.ScalarResources == nil {
		r.ScalarResources = map[corev1.ResourceName]int64{}
	}
	r.ScalarResources[name] = quantity
}

// ResourceList returns a resource list of this resource.
func (r *Resource) ResourceList() corev1.ResourceList {
	result := corev1.ResourceList{}
	if r.MilliCPU > 0 {
		result[corev1.ResourceCPU] = *resource.NewMilliQuantity(r.MilliCPU, resource.DecimalSI)
	}
	if r.Memory > 0 {
		result[corev1.ResourceMemory] = *resource.NewQuantity(r.Memory, resource.BinarySI)
	}
	if r.EphemeralStorage > 0 {
		result[corev1.ResourceEphemeralStorage] = *resource.NewQuantity(r.EphemeralStorage, resource.BinarySI)
	}
	if r.AllowedPodNumber > 0 {
		result[corev1.ResourcePods] = *resource.NewQuantity(r.AllowedPodNumber, resource.DecimalSI)
	}
	for rName, rQuant := range r.ScalarResources {
		if rQuant > 0 {
			if resourcehelper.IsHugePageResourceName(rName) {
				result[rName] = *resource.NewQuantity(rQuant, resource.BinarySI)
			} else {
				result[rName] = *resource.NewQuantity(rQuant, resource.DecimalSI)
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// AddPodRequest add the effective request resource of a pod to the origin resource.
// The Pod's effective request is the higher of:
// - the sum of all app containers(spec.Containers) request for a resource.
// - the effective init containers(spec.InitContainers) request for a resource.
// The effective init containers request is the highest request on all init containers.
func (r *Resource) AddPodRequest(podSpec *corev1.PodSpec) *Resource {
	for _, container := range podSpec.Containers {
		r.Add(container.Resources.Requests)
	}
	for _, container := range podSpec.InitContainers {
		r.SetMaxResource(container.Resources.Requests)
	}
	return r
}

// AddResourcePods adds pod resources into the Resource.
// Notice that a pod request resource list does not contain a request for pod resources,
// this function helps to add the pod resources.
func (r *Resource) AddResourcePods(pods int64) {
	r.Add(corev1.ResourceList{
		corev1.ResourcePods: *resource.NewQuantity(pods, resource.DecimalSI),
	})
}

// MinInt64 returns the smaller of two int64 numbers.
func MinInt64(a, b int64) int64 {
	if a <= b {
		return a
	}
	return b
}

// GetPodContainer get all container's resources status in a pod.
func GetPodContainer(pod *corev1.Pod, containerArg string) int32 {
	var result int32
	switch containerArg {
	case constants.RestartCount:
		var count int32
		containers := pod.Status.ContainerStatuses
		for _, container := range containers {
			count = container.RestartCount + count
		}
		result = count
	case constants.CPURequest:
		var cpuReq int64
		containers := pod.Spec.Containers
		for _, container := range containers {
			cpuReq = container.Resources.Requests.Cpu().MilliValue() + cpuReq
		}
		// CPU, in cores. (1000m = 1 cores)
		// return CPU in m
		result = int32(cpuReq) // #nosec G115
	case constants.CPULimit:
		var cpuLimit int64
		containers := pod.Spec.Containers
		for _, container := range containers {
			cpuLimit = container.Resources.Limits.Cpu().MilliValue() + cpuLimit
		}
		result = int32(cpuLimit) // #nosec G115
	case constants.MemoryRequest:
		var memoryReq int64
		containers := pod.Spec.Containers
		for _, container := range containers {
			// Memory, in bytes. (1024 * 1024 * 1024),to guarantee precision return
			// MilliValue returns the value of ceil(q * 1000); this could overflow an int64
			// result = Mi * 100
			// if front end need Mi result/100 ;if front end need Gi result/1024/100
			memoryReq = (container.Resources.Requests.Memory().MilliValue() / 1024 / 1024) + memoryReq
		}
		result = int32(memoryReq) // #nosec G115
	case constants.MemoryLimit:
		var memoryLimit int64
		containers := pod.Spec.Containers
		for _, container := range containers {
			memoryLimit = (container.Resources.Limits.Memory().MilliValue() / 1024 / 1024) + memoryLimit
		}
		result = int32(memoryLimit) // #nosec G115
	}
	return result
}
