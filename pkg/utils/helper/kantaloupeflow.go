package helper

import (
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kfv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1"
)

func GetReadyKantaloupeflowNum(flows []kfv1alpha1.KantaloupeFlow) int32 {
	var ready int32
	for _, flow := range flows {
		if IsReadyKantaloupeflow(flow) {
			ready++
		}
	}
	return ready
}

func IsReadyKantaloupeflow(flow kfv1alpha1.KantaloupeFlow) bool {
	for _, cnd := range flow.Status.Conditions {
		if cnd.Type == kfv1alpha1.ConditionTypeAvailable {
			return cnd.Status == corev1.ConditionTrue
		}
	}
	return false
}
