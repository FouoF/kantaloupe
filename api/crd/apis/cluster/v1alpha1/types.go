package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClusterResourceKind is the kind for the Cluster resource
	ClusterResourceKind = "Cluster"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope="Cluster"
// +kubebuilder:printcolumn:JSONPath=`.status.kubernetesVersion`,name="Version",type=string
// +kubebuilder:printcolumn:JSONPath=`.status.conditions[?(@.type=="Ready")].status`,name="Running",type=string
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired behavior of Cluster.
	Spec ClusterSpec `json:"spec"`

	// Status represents the most recently observed status of the Cluster.
	// +optional
	Status ClusterStatus `json:"status,omitempty"`
}

// ClusterSpec is the spec for a Cluster resource
type ClusterSpec struct {
	// Provider represents the cloud provider name of the member cluster.
	// +required
	Provider string `json:"provider"`

	// Type represents the type of the ai cluster.
	// +required
	Type string `json:"type"`

	// The API endpoint of the member cluster. This can be a hostname,
	// hostname:port, IP or IP:port.
	// +optional
	APIEndpoint string `json:"apiEndpoint,omitempty"`

	// SecretRef represents the secret contains mandatory credentials to access the member cluster.
	// The secret should hold credentials as follows:
	// - secret.data.token
	// - secret.data.caBundle
	// +optional
	SecretRef *LocalSecretReference `json:"secretRef,omitempty"`

	// PrometheusAddress represents the address of prometheus server.
	// +optional
	PrometheusAddress string `json:"prometheusAddress,omitempty"`

	// GatewayAddress represents the address of gateway server.
	// +optional
	GatewayAddress string `json:"gatewayAddress,omitempty"`

	// ClusterID represents the uuid of the cluster.
	ClusterId string `json:"clusterId"`
}

// LocalSecretReference is a reference to a secret within the enclosing
// namespace.
type LocalSecretReference struct {
	// Namespace is the namespace for the resource being referenced.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name is the name of resource being referenced.
	// kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// Define valid conditions of a member cluster.
const (
	// ClusterConditionReady means the cluster is healthy and ready to accept workloads.
	ClusterConditionReady = "Ready"
	// ClusterConditionModuleReady means the cluster moduler is healthy and ready to accept workloads.
	// e.g: gpu-operator and hami.
	ClusterConditionModuleReady = "ModuleReady"
)

// ClusterStatus is the status for a Cluster resource
type ClusterStatus struct {
	// KubernetesVersion represents version of the member cluster.
	// +optional
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`

	// KubeSystemId represents the uuid of sub cluster kube-system namespace.
	// +optional
	KubeSystemID string `json:"kubeSystemId,omitempty"`

	// NodeSummary represents the summary of nodes status in the member cluster.
	// +optional
	NodeSummary *ResourceSummary `json:"nodeSummary,omitempty"`

	// +optional
	PodSetSummary *ResourceSummary `json:"podSetSummary,omitempty"`

	// +optional
	KantaloupeflowSummary *ResourceSummary `json:"kantaloupeflowSummary,omitempty"`

	// +optional
	ResourceSummary *ClusterResourceSummary `json:"resourceSummary,omitempty"`

	// Conditions is an array of current conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ResourceSummary represents the summary of workload status in a specific cluster.
type ResourceSummary struct {
	// TotalNum is the total number of workloads in the cluster.
	// +optional
	TotalNum int32 `json:"totalNum,omitempty"`
	// ReadyNum is the number of ready workloads in the cluster.
	// +optional
	ReadyNum int32 `json:"readyNum,omitempty"`
}

// ResourceSummary represents the summary of resources in the member cluster.
type ClusterResourceSummary struct {
	// Allocatable represents the resources of a cluster that are available for scheduling.
	// Total amount of allocatable resources on all nodes.
	// +optional
	Allocatable corev1.ResourceList `json:"allocatable,omitempty"`
	// Allocating represents the resources of a cluster that are pending for scheduling.
	// Total amount of required resources of all Pods that are waiting for scheduling.
	// +optional
	Allocating corev1.ResourceList `json:"allocating,omitempty"`
	// Allocated represents the resources of a cluster that have been scheduled.
	// Total amount of required resources of all Pods that have been scheduled to nodes.
	// +optional
	Allocated corev1.ResourceList `json:"allocated,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList contains a list of container instances.
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items holds a list of Cluster.
	Items []Cluster `json:"items"`
}
