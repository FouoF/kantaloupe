package constants

import (
	"fmt"
	"strings"

	corev1alpha1 "github.com/dynamia-ai/kantaloupe/api/core/v1alpha1"
)

const (
	SelectAll = "__ALL__"
	// Label Type.
	PVCTypeLabelKey = "kantaloupe.dynamia.ai/pvc-type"

	// ClusterAliasAnnotationKey defines the annotation key of cluster alias.
	ClusterAliasAnnotationKey = "kantaloupe.dynamia.ai/alias-name"
	// ClusterDescribeAnnotationKey defines the annotation key of cluster describe.
	ClusterDescriptionAnnotationKey = "kantaloupe.dynamia.ai/description"
	// ClusterNameLableKey defines the lable key of cluster name.
	ClusterNameLableKey = "kantaloupe.dynamia.ai/name"

	// ManagedByLabelKey defines the label key for resources managed by kantaloupe.
	ManagedByLabelKey = "app.kubernetes.io/managed-by"
	// ManagedByLabelValue defines the label value for resources managed by kantaloupe.
	ManagedByLabelValue = "kantaloupe"

	// CredentialTypeLabelKey defines the label key for credential type.
	CredentialTypeLabelKey = "kantaloupe.io/credential-type" // #nosec G101 - not a credential
	// CredentialTypeDockerRegistry defines the label value for Docker registry credentials.
	CredentialTypeDockerRegistry = "docker-registry" // #nosec G101 - not a credential

	PodSecurityLabelTemplate = "pod-security.kubernetes.io/%s"

	StorageTypeKey = "kantaloupe.io/storage-type"

	HamiRegisterAnonationKey = "hami.io/node-nvidia-register"

	// TODO: fix misspelling.
	KantaloupeFlowAppLabelKey = "katanloupeflow-name"

	GlobalCluster          = "local-cluster"
	DefaultProvider        = "GENERIC"
	NodeSchedulingDisabled = "SchedulingDisabled"

	ControllerLabelKey = "node-role.kubernetes.io/control-plane"
	MasterLabelKey     = "node-role.kubernetes.io/master"
	WorkerLabelKey     = "node-role.kubernetes.io/worker"

	SortByDesc = "desc"
)

const (
	KindDeployment                  string = "Deployment"
	KindStatefulset                 string = "StatefulSet"
	KindDaemonset                   string = "DaemonSet"
	KindPod                         string = "Pod"
	KindService                     string = "Service"
	KindIngress                     string = "Ingress"
	KindJob                         string = "Job"
	KindCronJob                     string = "CronJob"
	KindReplicaSet                  string = "ReplicaSet"
	KindNetworkPolicy               string = "NetworkPolicy"
	KindHorizontalPodAutoscaler     string = "HorizontalPodAutoscaler"
	KindCronHorizontalPodAutoscaler string = "CronHPA"
	KindPersistentVolumeClaim       string = "PersistentVolumeClaim"
	KindGroupVersionResource        string = "GroupVersionResource"
)

// ServiceMonitor template names.
const (
	NvidiaSMTemplateName   = "nvidia-dcgm-exporter"
	MetaxSMTemplateName    = "metax-mx-exporter-monitor"
	AscendSMTemplateName   = "npu-exporter"
	HamiSchedulerSMName    = "hami-scheduler"
	HamiDevicePluginSMName = "hami-device-plugin"
)

// GPU resources.
const (
	NvidiaGPU            = "nvidia.com/gpu"
	NvidiaGPUCores       = "nvidia.com/gpucores"
	NvidiaGPUMemory      = "nvidia.com/gpumem"
	NvidiaQuotaGpu       = "requests.nvidia.com/gpucores"
	NvidiaQuotaGpuMemory = "requests.nvidia.com/gpumem"
	MetaxGPU             = "metax-tech.com/sgpu"
	MetaxGPUMemory       = "metax-tech.com/sgpumem"
	AWSNeuron            = "aws.amazon.com/neuron"
	AWSNeuronCore        = "aws.amazon.com/neuroncore"
)

// Provider constants.
const (
	GCPLabelKey = "cloud.google.com/gke-nodepool"
	AWSLabelKey = "eks.amazonaws.com/nodegroup"
)

var (
	PodSecurityLabelEnforce = fmt.Sprintf(PodSecurityLabelTemplate, corev1alpha1.Mode_enforce)
	PodSecurityLabelAudit   = fmt.Sprintf(PodSecurityLabelTemplate, corev1alpha1.Mode_audit)
	PodSecurityLabelWarn    = fmt.Sprintf(PodSecurityLabelTemplate, corev1alpha1.Mode_warn)

	JupyterServiceName = "jupyter"
	SSHServiceName     = "sshd"
	VSCodeServiceName  = "vscode"

	LocalClusterName = "local-cluster"

	HTTPProtocol  = "http"
	TCPProtocol   = "tcp"
	UDPProtocol   = "udp"
	HTTPSProtocol = "https"

	NetworkHTTPRoute = "httproute"
	NetworkTCPRoute  = "tcproute"

	DefaultPortVSCode  int32 = 6666
	DefaultPortJupyter int32 = 5555
	DefaultPortSSH     int32 = 22

	// set to container instance.
	EnvSSHRootPasswordKey            = "ROOT_PASSWORD"
	EnvEnableJupyter                 = "ENABLE_JUPYTERLAB"
	EnvEnableVSCode                  = "ENABLE_CODE_SERVER"
	EnvJupyterToken                  = "JUPYTER_TOKEN" //nolint: gosec
	EnvLibCudaLogLevel               = "LIBCUDA_LOG_LEVEL"
	CleanupInactiveWorkloadThreshold = "CLEANUP_INACTIVE_WORKLOAD_THRESHOLD"
	GatewayEnvBaseURL                = "GATEEAY_BASE_URL"
	GatewayEndpointEnv               = "GATEEAY_ENDPOINT"
	InitImageEnvKeyName              = "INIT_IMAGE"
	GatewayEnvNamePortStart          = "GATEWAY_PORT_START"
	GatewayEnvNamePortCount          = "GATEWAY_PORT_COUNT"
	SkipCheckClusterKubesystemID     = "SKIP_CHECK_CLUSTER_KUBESYSTEMID"

	// pod resource type.
	RestartCount  = "restartCount"
	CPURequest    = "cpuRequest"
	CPULimit      = "cpuLimit"
	MemoryRequest = "memoryRequest"
	MemoryLimit   = "memoryLimit"

	// KubeConfigDefaultTokenTemplate kubeConfig as service account token.
	KubeConfigDefaultTokenTemplate = strings.TrimSpace(`
apiVersion: v1
kind: Config
clusters:
- name: kantaloupe-cluster
  cluster:
    certificate-authority-data: %s
    server: %s
contexts:
- name: kantaloupe-cluster
  context:
    cluster: kantaloupe-cluster
    user: kantaloupe-cluster
current-context: kantaloupe-cluster
users:
- name: kantaloupe-cluster
  user:
    token: %s
`)

	KubeConfigTemplate = strings.TrimSpace(`
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: %s
    server: %s
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
    client-certificate-data: %s
    client-key-data: %s
`)
)
