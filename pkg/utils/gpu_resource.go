package utils

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/dynamia-ai/kantaloupe/pkg/constants"
)

func RequestNeuronResources(ctr *corev1.Container) bool {
	_, ok := ctr.Resources.Limits[corev1.ResourceName(constants.AWSNeuron)]
	if !ok {
		_, ok = ctr.Resources.Limits[corev1.ResourceName(constants.AWSNeuronCore)]
	}
	return ok
}

func RequestNvidiaResources(ctr *corev1.Container) bool {
	_, resourceNameOK := ctr.Resources.Limits[corev1.ResourceName(constants.NvidiaGPU)]
	if resourceNameOK {
		return true
	}

	_, resourceCoresOK := ctr.Resources.Limits[corev1.ResourceName(constants.NvidiaGPUCores)]
	_, resourceMemOK := ctr.Resources.Limits[corev1.ResourceName(constants.NvidiaGPUMemory)]
	if resourceCoresOK || resourceMemOK {
		return true
	}

	return false
}
