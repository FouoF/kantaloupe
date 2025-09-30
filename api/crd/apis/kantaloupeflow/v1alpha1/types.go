package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// KantaloupeFlow is the kind for the KantaloupeFlow resource
	KantaloupeFlowResourceKind = "KantaloupeFlow"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=kantaloupeflows,scope=Namespaced,shortName=klf,categories={dynamia-io}
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].reason",description="The status of the container"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type KantaloupeFlow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired behavior of ContainerInstance.
	Spec KantaloupeFlowSpec `json:"spec"`

	// Status represents the most recently observed status of the ContainerInstance.
	// +optional
	Status KantaloupeFlowStatus `json:"status"`
}

type PluginType string

const (
	SSHPluginType     PluginType = "ssh"
	VscodePluginType  PluginType = "vscode"
	JupyterPluginType PluginType = "jupyter"
)

type KantaloupeFlowSpec struct {
	// Plugins represents the programs that need to be installed for the current workload.
	Plugins []PluginType `json:"plugins,omitempty"`

	// Replicas for the container. TrainingRuntime must have this container.
	Replicas *int32 `json:"replicas,omitempty"`

	// Specifies the pod that will be created for this PodSpec
	// when executing a deployment.
	// +optional
	Template corev1.PodTemplateSpec `json:"template,omitempty"`

	// Indicates that the kantaloupeflow is paused.
	// +optional
	Paused bool `json:"paused,omitempty" protobuf:"varint,7,opt,name=paused"`

	// Networking is the networking configuration for the container
	// +kubebuilder:validation:MinItems=1
	Networking []Networking `json:"networking,omitempty"`

	// DependOn is the depend on resources for the container
	DependOn []DependOn `json:"dependOn,omitempty"`

	// The wrokload type this kantaloupeflow managed.
	Workload string `json:"workload"`
}

type Networking struct {
	// Name is the name of the networking
	Name string `json:"name"`
	// Type is the type of the networking
	// +kubebuilder:validation:Enum=httproute;tcproute
	Type string `json:"type"`
	// Protocol is the protocol of the networking
	// +kubebuilder:validation:Enum=http;tcp
	Protocol string `json:"protocol,omitempty"`
	// Port is the port of the networking
	Port int32 `json:"port"`
	// URL is the URL of the networking
	URL string `json:"url,omitempty"`
}

// DependOnKind is the kind of the depend on resource
type DependOnKind string

const (
	// DependOnKindConfigMap is a ConfigMap depend on resource
	DependOnKindConfigMap DependOnKind = "ConfigMap"
	// DependOnKindSecret is a Secret depend on resource
	DependOnKindSecret DependOnKind = "Secret"
)

// ResourceReference is a reference within the enclosing
// namespace.
type ResourceReference struct {
	// Namespace is the namespace for the resource being referenced.
	Namespace string `json:"namespace"`

	// Name is the name of resource being referenced.
	Name string `json:"name"`
}

type Effect struct {
	// Type is the effect of the depend on resource, e.g. ssh, apt, yum, etc.
	Type string `json:"type"`
	// VolumeMounts is the mount of the effect ops
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}

type DependOn struct {
	// Kind is the kind of the depend on resource, e.g. ConfigMap, Secret, etc.
	Kind DependOnKind `json:"kind"`
	// ResourceRef is the reference to the resource in the namespace
	ResourceRef ResourceReference `json:"resourceRef"`
	// Effect is the effect of the depend on resource, e.g. ssh, apt, yum, etc.
	Effect Effect `json:"effect"`
}

const (
	// Available means the deployment is available, ie. at least the minimum available
	// replicas required are up and running for at least minReadySeconds.
	ConditionTypeAvailable = "Available"
)

// KantaloupeFlowStatus is the status for a KantaloupeFlow resource
type KantaloupeFlowStatus struct {
	// Total number of non-terminated pods targeted by this deployment (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas,omitempty" protobuf:"varint,2,opt,name=replicas"`
	// readyReplicas is the number of pods targeted by this Deployment with a Ready Condition.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty" protobuf:"varint,7,opt,name=readyReplicas"`
	// Networking
	Networking []Networking `json:"networking,omitempty"`
	// Conditions is an array of current conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KantaloupeFlowList contains a list of kantaloupe flow.
type KantaloupeFlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items holds a list of ContainerInstance.
	Items []KantaloupeFlow `json:"items"`
}
