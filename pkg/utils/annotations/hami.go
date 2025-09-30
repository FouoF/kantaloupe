package annotations

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	PodGPUMemoryAnnotation  = "hami.io/vgpu-devices-allocated"
	OOMExpansionAnnotation  = "hami.io/nvidia-initial-device-memory"
	MemoryScaleAnnotation   = "NVIDIA_GPU_MEMORY_FACTOR"
	NodeNVIDIAGPUAnnotation = "hami.io/node-nvidia-register"
	PodNeuronsAnnotation    = "hami.io/aws-neuron-devices-allocated"
)

type GPUAllocation struct {
	UUID   string
	Vendor string
	Memory int64
	Core   int32
}

func MarshalGPUAllocationAnnotation(annotation string) ([]*GPUAllocation, error) {
	res := []*GPUAllocation{}
	allocations := strings.SplitSeq(annotation, ":")
	for allocation := range allocations {
		values := strings.Split(allocation, ",")
		if values[0] == ";" {
			break
		}
		if len(values) != 4 {
			return nil, errors.New("invalid format for annotation: " + allocation)
		}
		memory, err := strconv.Atoi(values[2])
		if err != nil {
			return nil, err
		}
		core, err := strconv.ParseInt(values[3], 10, 32)
		if err != nil {
			return nil, err
		}
		res = append(res, &GPUAllocation{
			UUID:   values[0],
			Vendor: values[1],
			Memory: int64(memory),
			Core:   int32(core),
		})
	}
	return res, nil
}

func UnmarshalGPUAllocationAnnotation(annotations []*GPUAllocation) string {
	s := ""
	for _, allocation := range annotations {
		t := fmt.Sprintf("%s,%s,%d,%d:", allocation.UUID, allocation.Vendor, allocation.Memory, allocation.Core)
		s += t
	}
	return s + ";"
}

func GetFactorFromAnnotation(annotations map[string]string) float64 {
	factorStr, ok := annotations[MemoryScaleAnnotation]
	if !ok {
		return 1.0
	}
	factor, err := strconv.ParseFloat(factorStr, 64)
	if err != nil || factor == 0.0 {
		return 1.0
	}
	return factor
}
