package pod

import (
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/dynamia-ai/kantaloupe/pkg/engine"
)

type podInfo struct {
	Namespace string
	Name      string
	UID       k8stypes.UID
	NodeID    string
	CtrIDs    []string
}

type podManager struct {
	pods  map[k8stypes.UID]*podInfo
	mutex sync.RWMutex
}

type Service struct {
	podManager
	clientManager engine.ClientManagerInterface
	podLister     listerscorev1.PodLister
}

func (m *Service) OnAddPod(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		klog.ErrorS(fmt.Errorf("invalid pod object"), "Failed to process pod addition")
		return
	}
	klog.V(5).InfoS("Pod added", "pod", pod.Name, "namespace", pod.Namespace)
	if IsPodInTerminatedState(pod) {
		m.delPod(pod)
		return
	}
	m.addPod(pod, pod.Spec.NodeName)
}

func (m *Service) OnUpdatePod(_, newObj interface{}) {
	m.OnAddPod(newObj)
}

func (m *Service) OnDelPod(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		klog.Errorf("unknown add object type")
		return
	}
	m.delPod(pod)
}

func (m *Service) addPod(pod *corev1.Pod, nodeID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.pods[pod.UID]
	if !exists {
		pi := &podInfo{
			Name:      pod.Name,
			UID:       pod.UID,
			Namespace: pod.Namespace,
			NodeID:    nodeID,
		}
		m.pods[pod.UID] = pi
		klog.V(4).InfoS("Pod added",
			"pod", klog.KRef(pod.Namespace, pod.Name),
			"nodeID", nodeID,
		)
	} else {
		klog.V(4).InfoS("Pod devices updated",
			"pod", klog.KRef(pod.Namespace, pod.Name),
		)
	}
}

func (m *Service) delPod(pod *corev1.Pod) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	pi, exists := m.pods[pod.UID]
	if exists {
		klog.InfoS("Pod deleted",
			"pod", klog.KRef(pod.Namespace, pod.Name),
			"nodeID", pi.NodeID,
		)
		delete(m.pods, pod.UID)
	} else {
		klog.InfoS("Pod not found for deletion",
			"pod", klog.KRef(pod.Namespace, pod.Name),
		)
	}
}

func IsPodInTerminatedState(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded
}

func NewService(pL listerscorev1.PodLister, clientManager engine.ClientManagerInterface) *Service {
	return &Service{
		clientManager: clientManager,
		podManager: podManager{
			pods: make(map[k8stypes.UID]*podInfo),
		},
		podLister: pL,
	}
}
