package bff

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	resourceapi "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clustersv1alpha1 "github.com/dynamia-ai/kantaloupe/api/clusters/v1alpha1"
	flowcrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1"
	flowv1alpha1 "github.com/dynamia-ai/kantaloupe/api/kantaloupeflow/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/api/types"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
)

// ConvertProto2Kantaloupeflow converts kantaloupeflow cr to protobuf.
func ConvertProto2Kantaloupeflow(flow *flowv1alpha1.Kantaloupeflow) *flowcrdv1alpha1.KantaloupeFlow {
	if flow == nil {
		return &flowcrdv1alpha1.KantaloupeFlow{}
	}

	res := &flowcrdv1alpha1.KantaloupeFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:        flow.Metadata.Name,
			Namespace:   flow.Metadata.Namespace,
			Labels:      flow.Metadata.Labels,
			Annotations: flow.Metadata.Annotations,
		},
	}
	if flow.Spec != nil {
		res.Spec = flowcrdv1alpha1.KantaloupeFlowSpec{
			Plugins:  ConvertProto2PluginType(flow.Spec.Plugins),
			Replicas: &flow.Spec.Replicas,
			Template: ConvertProto2PodTemplate(flow.Spec.Template),
			Paused:   flow.Spec.Paused,
			Workload: ConvertProto2Workload(&flow.Spec.Workload),
		}
	}

	return res
}

func ConvertProto2PluginType(plugins []flowv1alpha1.PluginType) []flowcrdv1alpha1.PluginType {
	res := []flowcrdv1alpha1.PluginType{}
	for _, plugin := range plugins {
		res = append(res, flowcrdv1alpha1.PluginType(plugin.String()))
	}
	return res
}

func ConvertProto2PodTemplate(template *flowv1alpha1.PodTemplateSpec) corev1.PodTemplateSpec {
	if template == nil {
		return corev1.PodTemplateSpec{}
	}

	objectMeta := metav1.ObjectMeta{}
	if template.Metadata != nil {
		objectMeta.Name = template.Metadata.Name
		objectMeta.Namespace = template.Metadata.Namespace
		objectMeta.Labels = template.Metadata.Labels
		objectMeta.Annotations = template.Metadata.Annotations
	}

	res := corev1.PodTemplateSpec{
		ObjectMeta: objectMeta,
		Spec: corev1.PodSpec{
			Volumes:    ConvertProto2Volumes(template.Spec.Volumes),
			Containers: ConvertProto2Container(template.Spec.Containers),
		},
	}

	return res
}

func ConvertProto2Container(containers []*flowv1alpha1.Container) []corev1.Container {
	res := []corev1.Container{}

	for _, container := range containers {
		// convert container port.
		ports := []corev1.ContainerPort{}
		for _, port := range container.Ports {
			ports = append(ports, corev1.ContainerPort{
				ContainerPort: port.ContainerPort,
				HostPort:      port.HostPort,
				Name:          port.Name,
				Protocol:      corev1.Protocol(port.Protocol),
			})
		}

		volumeMounts := []corev1.VolumeMount{}
		for _, vm := range container.VolumeMounts {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      vm.Name,
				MountPath: vm.MountPath,
				SubPath:   vm.SubPath,
				ReadOnly:  vm.ReadOnly,
			})
		}

		env := []corev1.EnvVar{}
		if container.Env != nil {
			for _, e := range container.Env {
				env = append(env, corev1.EnvVar{
					Name:  e.Name,
					Value: e.Value,
				})
			}
		}

		v1container := corev1.Container{
			Name:            container.Name,
			Image:           container.Image,
			WorkingDir:      container.WorkingDir,
			Ports:           ports,
			ImagePullPolicy: corev1.PullPolicy(container.ImagePullPolicy),
			Resources:       ConvertProto2ResourceRequirements(container.Resources),
			VolumeMounts:    volumeMounts,
		}

		if len(container.Command) > 0 {
			v1container.Command = container.Command
		}
		if len(container.Args) > 0 {
			v1container.Args = container.Args
		}
		if len(env) > 0 {
			v1container.Env = env
		}

		res = append(res, v1container)
	}

	return res
}

func ConvertProto2ResourceRequirements(resource *flowv1alpha1.ResourceRequirements) corev1.ResourceRequirements {
	if resource == nil {
		return corev1.ResourceRequirements{}
	}
	// If specify limit and not specify request for a resource, copy the limit to the request.
	for k, v := range resource.Limits.Resources {
		if resource.Requests.Resources == nil {
			resource.Requests.Resources = make(map[string]string)
		}
		if _, ok := resource.Requests.Resources[k]; !ok {
			resource.Requests.Resources[k] = v
		}
	}

	return corev1.ResourceRequirements{
		Limits:   ConvertProto2ResourceList(resource.Limits),
		Requests: ConvertProto2ResourceList(resource.Requests),
	}
}

func ConvertProto2ResourceList(resource *flowv1alpha1.ResourceList) corev1.ResourceList {
	res := corev1.ResourceList{}
	if resource == nil {
		return res
	}

	if resource.Cpu != "" {
		res[corev1.ResourceCPU] = (resourceapi.MustParse(resource.Cpu))
	}
	if resource.Memory != "" {
		res[corev1.ResourceMemory] = (resourceapi.MustParse(resource.Memory))
	}
	if resource.Storage != "" {
		res[corev1.ResourceStorage] = (resourceapi.MustParse(resource.Storage))
	}

	for k, v := range resource.Resources {
		res[corev1.ResourceName(k)] = resourceapi.MustParse(v)
	}

	return res
}

func ConvertProto2Volumes(volumes []*flowv1alpha1.Volume) []corev1.Volume {
	res := []corev1.Volume{}

	for _, volume := range volumes {
		// convert secret source.
		var secret *corev1.SecretVolumeSource
		if volume.Secret != nil {
			secret = &corev1.SecretVolumeSource{
				SecretName:  volume.Secret.SecretName,
				DefaultMode: &volume.Secret.DefaultMode,
				Optional:    &volume.ConfigMap.Optional,
				Items:       ConvertProto2KeyToPath(volume.Secret.Items),
			}
		}

		// convert hostPath source.
		var hostPath *corev1.HostPathVolumeSource
		if volume.HostPath != nil {
			hostPath = &corev1.HostPathVolumeSource{
				Type: (*corev1.HostPathType)(&volume.HostPath.Type),
				Path: volume.HostPath.Path,
			}
		}

		// convert configmap source.
		var configMap *corev1.ConfigMapVolumeSource
		if volume.ConfigMap != nil {
			configMap = &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: volume.ConfigMap.Name},
				Items:                ConvertProto2KeyToPath(volume.ConfigMap.Items),
				DefaultMode:          &volume.ConfigMap.DefaultMode,
				Optional:             &volume.ConfigMap.Optional,
			}
		}

		// convert persistentVolumeClaim source.
		var pvc *corev1.PersistentVolumeClaimVolumeSource
		if volume.PersistentVolumeClaim != nil {
			pvc = &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: volume.PersistentVolumeClaim.ClaimName,
				ReadOnly:  volume.PersistentVolumeClaim.ReadOnly,
			}
		}

		res = append(res, corev1.Volume{
			Name: volume.Name,
			VolumeSource: corev1.VolumeSource{
				Secret:                secret,
				PersistentVolumeClaim: pvc,
				HostPath:              hostPath,
				ConfigMap:             configMap,
			},
		})
	}

	return res
}

func ConvertProto2KeyToPath(keyToPaths []*flowv1alpha1.KeyToPath) []corev1.KeyToPath {
	res := []corev1.KeyToPath{}
	for _, ktp := range keyToPaths {
		res = append(res, corev1.KeyToPath{
			Key:  ktp.Key,
			Path: ktp.Path,
			Mode: &ktp.Mode,
		})
	}
	return res
}

func ConvertKantaloupeflows2Proto(flows []*flowcrdv1alpha1.KantaloupeFlow) []*flowv1alpha1.Kantaloupeflow {
	res := []*flowv1alpha1.Kantaloupeflow{}
	for _, flow := range flows {
		res = append(res, ConvertKantaloupeflow2Proto(flow))
	}

	return res
}

func ConvertKantaloupeflow2Proto(flow *flowcrdv1alpha1.KantaloupeFlow) *flowv1alpha1.Kantaloupeflow {
	if flow == nil {
		return &flowv1alpha1.Kantaloupeflow{}
	}

	return &flowv1alpha1.Kantaloupeflow{
		Metadata: convertObjectMeta(flow.ObjectMeta),
		Spec:     convertSpec2Proto(flow.Spec),
		Status:   convertStatus2Proto(flow.Status),
	}
}

func convertStatus2Proto(status flowcrdv1alpha1.KantaloupeFlowStatus) *flowv1alpha1.KantaloupeflowStatus {
	networks := []*flowv1alpha1.Network{}
	for _, network := range status.Networking {
		networks = append(networks, &flowv1alpha1.Network{
			Name: network.Name,
			Url:  network.URL,
		})
	}

	res := &flowv1alpha1.KantaloupeflowStatus{
		Replicas:      status.Replicas,
		ReadyReplicas: status.ReadyReplicas,
		Conditions:    convertConditions(status.Conditions),
		Networks:      networks,
		State:         calculateKantaloupeflowState(status.Conditions),
	}

	return res
}

func calculateKantaloupeflowState(condition []metav1.Condition) flowv1alpha1.KantaloupeflowState {
	available := utils.GetConditionByType(condition, string(appsv1.DeploymentAvailable))
	progressing := utils.GetConditionByType(condition, string(appsv1.DeploymentProgressing))
	if available != nil {
		if available.Status == metav1.ConditionTrue {
			return flowv1alpha1.KantaloupeflowState_Running
		}
		if progressing != nil {
			if progressing.Status == metav1.ConditionTrue {
				return flowv1alpha1.KantaloupeflowState_Progressing
			}
		}
		if available.Status == metav1.ConditionFalse {
			return flowv1alpha1.KantaloupeflowState_Falied
		}
	}

	return flowv1alpha1.KantaloupeflowState_Progressing
}

func convertConditions(condition []metav1.Condition) []*types.Condition {
	result := make([]*types.Condition, len(condition))
	for index, v := range condition {
		result[index] = &types.Condition{
			Message:            v.Message,
			Reason:             v.Reason,
			LastTransitionTime: v.LastTransitionTime.Format("2006-01-02 15:04:05"),
			Status:             string(v.Status),
			Type:               v.Type,
		}
	}
	return result
}

func convertSpec2Proto(spec flowcrdv1alpha1.KantaloupeFlowSpec) *flowv1alpha1.KantaloupeflowSpec {
	// convert pluginType.
	plugins := []flowv1alpha1.PluginType{}
	for _, plugin := range spec.Plugins {
		v := flowv1alpha1.PluginType_value[string(plugin)]
		plugins = append(plugins, flowv1alpha1.PluginType(v))
	}

	containers := []*flowv1alpha1.Container{}
	for _, container := range spec.Template.Spec.Containers {
		envs := []*flowv1alpha1.EnvVar{}
		for _, env := range container.Env {
			envs = append(envs, &flowv1alpha1.EnvVar{
				Name:  env.Name,
				Value: env.Value,
			})
		}

		containers = append(containers, &flowv1alpha1.Container{
			Name:      container.Name,
			Image:     container.Image,
			Env:       envs,
			Resources: convertResourceToProto(container.Resources),
		})
	}

	tmp := &flowv1alpha1.PodTemplateSpec{
		Metadata: convertObjectMeta(spec.Template.ObjectMeta),
		Spec: &flowv1alpha1.PodSpec{
			Containers: containers,
		},
	}

	return &flowv1alpha1.KantaloupeflowSpec{
		Plugins:  plugins,
		Replicas: *spec.Replicas,
		Template: tmp,
		Paused:   spec.Paused,
	}
}

func convertResourceToProto(resource corev1.ResourceRequirements) *flowv1alpha1.ResourceRequirements {
	convertFunc := func(resources corev1.ResourceList) *flowv1alpha1.ResourceList {
		r := &flowv1alpha1.ResourceList{
			Resources: map[string]string{},
		}
		for key, val := range resources {
			switch key {
			case corev1.ResourceCPU:
				r.Cpu = val.String()
			case corev1.ResourceMemory:
				r.Memory = val.String()
			case corev1.ResourceStorage:
				r.Storage = val.String()
			default:
				r.Resources[key.String()] = val.String()
			}
		}
		return r
	}

	return &flowv1alpha1.ResourceRequirements{
		Limits:   convertFunc(resource.Limits),
		Requests: convertFunc(resource.Requests),
	}
}

func ConvertProto2Workload(workload *flowv1alpha1.WorkloadType) string {
	if workload == nil || *workload == flowv1alpha1.WorkloadType_Pod {
		return "pod"
	}
	return "deployment"
}

func patchByProvider(flow *flowcrdv1alpha1.KantaloupeFlow, provider string) error {
	switch provider {
	case clustersv1alpha1.ClusterProvider_AWS_EKS.String():
		return patchAWSEKSCluster(flow)
	case clustersv1alpha1.ClusterProvider_GCP_GKE.String():
		return patchGCPGKECluster(flow)
	default:
		return nil
	}
}

func patchAWSEKSCluster(flow *flowcrdv1alpha1.KantaloupeFlow) error {
	flow.Spec.Template.Spec.SchedulerName = "hami-scheduler"
	for i := range flow.Spec.Template.Spec.Containers {
		if utils.RequestNeuronResources(&flow.Spec.Template.Spec.Containers[i]) {
			flow.Spec.Template.Spec.Containers[i].Env = append(flow.Spec.Template.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  "NEURON_PROCESS_TAG",
				Value: fmt.Sprintf("%s/%s", flow.Namespace, flow.Name),
			})
		}
	}
	return nil
}

func patchGCPGKECluster(flow *flowcrdv1alpha1.KantaloupeFlow) error {
	for i := range flow.Spec.Template.Spec.Containers {
		if utils.RequestNvidiaResources(&flow.Spec.Template.Spec.Containers[i]) {
			flow.Spec.Template.Spec.Tolerations = append(flow.Spec.Template.Spec.Tolerations, corev1.Toleration{
				Key:      "nvidia.com/gpu",
				Operator: "Equal",
				Value:    "present",
				Effect:   "NoSchedule",
			})
			return nil
		}
	}
	return nil
}
