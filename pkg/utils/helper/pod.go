package helper

import (
	corev1 "k8s.io/api/core/v1"
)

func GetReadyPodNum(pods []*corev1.Pod) int32 {
	var ready int32
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodRunning && hasPodReadyCondition(pod.Status.Conditions) {
			ready++
		}
	}
	return ready
}

func GetNodeNonTerminatedPodsList(pods []*corev1.Pod) []*corev1.Pod {
	result := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			result = append(result, pod)
		}
	}
	return result
}

func hasPodReadyCondition(conditions []corev1.PodCondition) bool {
	for _, condition := range conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
