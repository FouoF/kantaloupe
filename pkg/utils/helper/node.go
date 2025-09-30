package helper

import corev1 "k8s.io/api/core/v1"

// NodeReady checks whether the node condition is ready.
func NodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// GpuNode checks whether the node is gpu node.
func GpuNode(node *corev1.Node) bool {
	if len(node.Labels) > 0 {
		if _, ok := node.Labels["gpu"]; ok {
			return true
		}
	}
	return false
}
