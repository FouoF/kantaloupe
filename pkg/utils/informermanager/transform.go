package informermanager

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

var (
	NodeGVR        = corev1.SchemeGroupVersion.WithResource("nodes")
	PodGVR         = corev1.SchemeGroupVersion.WithResource("pods")
	NamespaceGVR   = corev1.SchemeGroupVersion.WithResource("namespaces")
	DeploymentGVR  = appsv1.SchemeGroupVersion.WithResource("deployments")
	StatefulSetGVR = appsv1.SchemeGroupVersion.WithResource("statefulsets")
	DaemonSetGVR   = appsv1.SchemeGroupVersion.WithResource("daemonsets")

	TransformFuns = map[schema.GroupVersionResource]cache.TransformFunc{
		NodeGVR:        NodeTransformFunc,
		PodGVR:         PodTransformFunc,
		DeploymentGVR:  DeploymentTransformFunc,
		StatefulSetGVR: StatefulSetTransformFunc,
		DaemonSetGVR:   DaemonSetTransformFunc,
	}
)

func PodTransformFunc(obj interface{}) (interface{}, error) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return obj, nil
	}
	u, _ = RemoveCommonFields(u)
	removedFields := [][]string{
		{"metadata", "generateName"},
		{"metadata", "selfLink"},
		{"metadata", "generation"},
		{"spec", "volumes"},
		{"spec", "ephemeralContainers"},
		{"spec", "restartPolicy"},
		{"spec", "dnsPolicy"},
		{"spec", "nodeSelector"},
		{"spec", "serviceAccountName"},
		{"spec", "serviceAccount"},
		{"spec", "hostNetwork"},
		{"spec", "hostPID"},
		{"spec", "hostIPC"},
		{"spec", "imagePullSecrets"},
		{"spec", "hostname"},
		{"spec", "subdomain"},
		{"spec", "schedulerName"},
		{"spec", "tolerations"},
		{"spec", "hostAliases"},
		{"spec", "priorityClassName"},
		{"spec", "readinessGates"},
		{"spec", "topologySpreadConstraints"},
		{"spec", "schedulingGates"},
		{"spec", "resourceClaims"},
		{"status", "message"},
		{"status", "reason"},
		{"status", "nominatedNodeName"},
		{"status", "hostIP"},
		{"status", "podIP"},
		{"status", "podIPs"},
		{"status", "initContainerStatuses"},
		{"status", "containerStatuses"},
		{"status", "qosClass"},
		{"status", "ephemeralContainerStatuses"},
	}
	for _, r := range removedFields {
		unstructured.RemoveNestedField(u.Object, r...)
	}
	return u, nil
}

func DeploymentTransformFunc(obj interface{}) (interface{}, error) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return obj, nil
	}
	u, _ = RemoveCommonFields(u)
	val, _, _ := unstructured.NestedFieldNoCopy(u.Object, "spec", "replicas")

	unstructured.RemoveNestedField(u.Object, "spec")
	unstructured.SetNestedField(u.Object, val, "spec", "replicas")
	return u, nil
}

func NodeTransformFunc(obj interface{}) (interface{}, error) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return obj, nil
	}
	u, _ = RemoveCommonFields(u)
	removedFields := [][]string{
		{"spec"},
		{"status", "images"},
		{"status", "addresses"},
		{"status", "nodeInfo"},
		{"status", "volumesInUse"},
		{"status", "volumesAttached"},
		{"status", "daemonEndpoints"},
	}
	for _, r := range removedFields {
		unstructured.RemoveNestedField(u.Object, r...)
	}
	return u, nil
}

func StatefulSetTransformFunc(obj interface{}) (interface{}, error) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return obj, nil
	}
	u, _ = RemoveCommonFields(u)
	val, _, _ := unstructured.NestedFieldNoCopy(u.Object, "spec", "replicas")
	removedFields := [][]string{
		{"spec"},
		{"status", "currentRevision"},
		{"status", "updateRevision"},
	}
	for _, r := range removedFields {
		unstructured.RemoveNestedField(u.Object, r...)
	}
	unstructured.SetNestedField(u.Object, val, "spec", "replicas")
	return u, nil
}

func DaemonSetTransformFunc(obj interface{}) (interface{}, error) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return obj, nil
	}
	u, _ = RemoveCommonFields(u)
	unstructured.RemoveNestedField(u.Object, "spec")
	return u, nil
}

func RemoveCommonFields(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	removedFields := [][]string{
		{"metadata", "uid"},
		{"metadata", "labels"},
		{"metadata", "annotations"},
		{"metadata", "creationTimestamp"},
		{"metadata", "managedFields"},
		{"metadata", "resourceVersion"},
		{"metadata", "finalizers"},
		{"metadata", "ownerReferences"},
	}
	for _, r := range removedFields {
		unstructured.RemoveNestedField(obj.Object, r...)
	}
	return obj, nil
}
