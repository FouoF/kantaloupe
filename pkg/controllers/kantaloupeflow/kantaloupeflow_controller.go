package kantaloupeflow

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	clustercrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
	kfv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/annotations"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/env"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/helper"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/namespace"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/portallocate"
)

const (
	KantaloupeControllerFinalizer = "kantaloupe.dynamia.io/kantaloupe-controller"

	ControllerName = "%s-kantaloupeflow-controller"

	PodAllocationAnnotation = "kantaloupe.dynamia.io/pod-allocation-meet"

	OOMExpansionAnnotation = "kantaloupe.dynamia.io/oom-expansion-to"
)

type Controller struct {
	Cluster string
	client.Client
	LocalClusterClient client.Client
	PortAllocate       portallocate.Allocate
	EventRecorder      record.EventRecorder
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *Controller) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.V(4).InfoS("Reconciling Cluster", "KObj", req.NamespacedName.String())

	flow := &kfv1alpha1.KantaloupeFlow{}
	if err := c.Client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, flow); err != nil {
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		return controllerruntime.Result{}, err
	}
	if err := c.ensureFinalizer(ctx, flow); err != nil {
		klog.ErrorS(err, "faild to ensure finalizer for kantaloupeFlow", "kantaloupeFlow", klog.KObj(flow))
		return controllerruntime.Result{}, err
	}

	if !flow.DeletionTimestamp.IsZero() {
		if err := c.removeFinalizer(ctx, flow); err != nil {
			klog.ErrorS(err, "failed to delete finalizer for kantaloupeFlow", "kantaloupeFlow", klog.KObj(flow))
			return controllerruntime.Result{}, err
		}
		return controllerruntime.Result{}, nil
	}

	return controllerruntime.Result{}, c.syncKantaloupeFlow(ctx, flow.DeepCopy())
}

func (c *Controller) syncKantaloupeFlow(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) error {
	if err := c.ensureNetworking(ctx, flow); err != nil {
		klog.ErrorS(err, "failed to sync kantaloupeFlow networking", "kantaloupeFlow", klog.KObj(flow))
		return err
	}

	// create apt resources and pip config configmaps for the kantaloupeflow.
	if err := c.createAptResourcesAndPipConfig(ctx, flow); err != nil {
		klog.ErrorS(err, "failed to create apt resources and pip config for kantaloupeFlow", "kantaloupeFlow", klog.KObj(flow))
		return err
	}

	if flow.Spec.Workload == "pod" {
		if _, err := c.ensurePod(ctx, flow); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
		}
		if err := c.syncKantaloupeFlowStatus(ctx, flow); err != nil {
			return err
		}
	} else {
		if err := c.ensureDeployment(ctx, flow); err != nil {
			klog.ErrorS(err, "failed to ensure deployment for kantaloupeFlow", "kantaloupeFlow", klog.KObj(flow))
			return err
		}
	}

	if err := c.ensureAnnotation(ctx, flow); err != nil {
		return err
	}

	return nil
}

func (c *Controller) ensureNetworking(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) error {
	// if there is not plugins, skip to create networking resource.
	if !isKantaloupeflowEabledPlugin(flow) {
		return nil
	}

	// 1. create service for plugins(vscode, jupyter, sshd).
	_, err := c.ensureService(ctx, flow)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	// FIXME: not use nodeport.
	// host, err := c.GetMasterNodeIP(ctx)
	// if err != nil {
	// 	return err
	// }

	// 2. create httproute and tcptoute, the cluster must be installed gateway-api.
	cluster := &clustercrdv1alpha1.Cluster{}
	if err := c.LocalClusterClient.Get(ctx, client.ObjectKey{Name: c.Cluster}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	networkings := []kfv1alpha1.Networking{}
	for i := range flow.Spec.Networking {
		network := flow.Spec.Networking[i]
		if network.Type == "httproute" {
			httpMatchURL, err := c.ensureHTTPRoute(ctx, flow, &network)
			if err != nil {
				klog.ErrorS(err, "failed to create httproute")
				return err
			}
			network.URL = fmt.Sprintf("%s%s", strings.TrimRight(cluster.Spec.GatewayAddress, "/"), httpMatchURL)
		}
		if network.Type == "tcproute" {
			port, err := c.ensureTCPRoute(ctx, flow, &network)
			if err != nil {
				klog.ErrorS(err, "failed to create tcproute")
				return err
			}

			u, err := url.Parse(cluster.Spec.GatewayAddress)
			if err != nil {
				return err
			}

			network.URL = fmt.Sprintf("%s:%d", u.Host, port)
		}

		networkings = append(networkings, network)
	}

	// 3. update the URL for kantaloupeflow.
	if !equality.Semantic.DeepEqual(flow.Status.Networking, networkings) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err := utils.UpdateStatus(ctx, c.Client, flow,
				func() error {
					flow.Status.Networking = networkings
					return nil
				})
			return err
		})
		if err != nil {
			klog.ErrorS(err, "Failed to update kantaloupeflow status", "kantaloupeflow", klog.KObj(flow))
			return err
		}
	}

	return nil
}

func (c *Controller) ensureTCPRoute(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow, network *kfv1alpha1.Networking) (int, error) {
	tcpRoute := &gatewayv1alpha2.TCPRoute{}
	tcpRoute.Name = fmt.Sprintf("%s-%s", flow.GetName(), network.Name)
	tcpRoute.Namespace = flow.GetNamespace()
	tcpRoute.Labels = labels.Merge(flow.GetLabels(), labels.Set{
		constants.KantaloupeFlowAppLabelKey: flow.GetName(),
	})

	tcpRoute.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: flow.APIVersion,
			Kind:       flow.Kind,
			Name:       flow.Name,
			UID:        flow.UID,
			Controller: ptr.To(true),
		},
	}

	// user pre allocated port.
	var allocatePort int
	var err error

	older := &gatewayv1alpha2.TCPRoute{}
	getErr := c.Get(ctx, client.ObjectKey{Namespace: tcpRoute.Namespace, Name: tcpRoute.Name}, older)
	if getErr != nil {
		if !apierrors.IsNotFound(getErr) {
			return -1, err
		}
		allocatePort, err = c.PortAllocate.AllocateNext()
		if err != nil {
			return -1, err
		}
	} else {
		if len(older.Spec.ParentRefs) == 0 {
			return -1, fmt.Errorf("no parent refs found for %s", klog.KObj(older))
		}
		sectionName := older.Spec.ParentRefs[0].SectionName
		_, allocatePort, err = portallocate.SplitGatewaySectionName(string(*sectionName))
		if err != nil {
			return -1, err
		}
	}

	tcpRoute.Spec.ParentRefs = []gatewayv1.ParentReference{
		{
			Name:        "kantaloupe",
			Namespace:   ptr.To(gatewayv1.Namespace(helper.GetCurrentNSOrDefault())),
			SectionName: ptr.To(gatewayv1.SectionName(generateGatewaySectionName(network, allocatePort))),
		},
	}
	tcpRoute.Spec.Rules = append(tcpRoute.Spec.Rules, gatewayv1alpha2.TCPRouteRule{
		BackendRefs: []gatewayv1alpha2.BackendRef{
			{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name: gatewayv1.ObjectName(flow.Name),
					Port: ptr.To(gatewayv1.PortNumber(network.Port)),
				},
			},
		},
	})

	if apierrors.IsNotFound(getErr) {
		return allocatePort, c.Create(ctx, tcpRoute)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		tcpRoute.ResourceVersion = older.ResourceVersion
		return c.Update(ctx, tcpRoute)
	})
	if retryErr != nil {
		return -1, fmt.Errorf("failed to update httproute: %w", retryErr)
	}

	return allocatePort, nil
}

func (c *Controller) ensureHTTPRoute(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow, network *kfv1alpha1.Networking) (string, error) {
	httpRoute := &gatewayv1.HTTPRoute{}
	httpRoute.Name = fmt.Sprintf("%s-%s", flow.GetName(), network.Name)
	httpRoute.Namespace = flow.GetNamespace()
	httpRoute.Labels = labels.Merge(flow.GetLabels(), labels.Set{
		constants.KantaloupeFlowAppLabelKey: flow.GetName(),
	})
	httpMatchURL := generateHTTPRouteMatchURL(network, flow)

	httpRoute.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: flow.APIVersion,
			Kind:       flow.Kind,
			Name:       flow.Name,
			UID:        flow.UID,
			Controller: ptr.To(true),
		},
	}
	httpRoute.Spec = gatewayv1.HTTPRouteSpec{
		CommonRouteSpec: gatewayv1.CommonRouteSpec{
			ParentRefs: []gatewayv1.ParentReference{
				{
					Name:      "kantaloupe",
					Namespace: ptr.To(gatewayv1.Namespace(helper.GetCurrentNSOrDefault())),
				},
			},
		},
	}
	httpRoute.Spec.Rules = append(httpRoute.Spec.Rules, gatewayv1.HTTPRouteRule{
		Matches: []gatewayv1.HTTPRouteMatch{
			{
				Path: &gatewayv1.HTTPPathMatch{
					Type:  ptr.To(gatewayv1.PathMatchPathPrefix),
					Value: ptr.To(httpMatchURL),
				},
			},
		},
		BackendRefs: []gatewayv1.HTTPBackendRef{
			{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: gatewayv1.ObjectName(flow.Name),
						Port: ptr.To(gatewayv1.PortNumber(network.Port)),
					},
				},
			},
		},
	})
	if network.Name == constants.VSCodeServiceName {
		httpRoute.Spec.Rules[len(httpRoute.Spec.Rules)-1].Filters = []gatewayv1.HTTPRouteFilter{
			{
				Type: gatewayv1.HTTPRouteFilterURLRewrite,
				URLRewrite: &gatewayv1.HTTPURLRewriteFilter{
					Path: &gatewayv1.HTTPPathModifier{
						Type:               gatewayv1.PrefixMatchHTTPPathModifier,
						ReplacePrefixMatch: ptr.To("/"),
					},
				},
			},
		}
		// add the vscode path suffix.
		path := *httpRoute.Spec.Rules[len(httpRoute.Spec.Rules)-1].Matches[0].Path.Value
		path += "/"
		httpRoute.Spec.Rules[len(httpRoute.Spec.Rules)-1].Matches[0].Path.Value = ptr.To(path)

		httpMatchURL = path
	}

	older := &gatewayv1.HTTPRoute{}
	err := c.Get(ctx, client.ObjectKey{Namespace: httpRoute.Namespace, Name: httpRoute.Name}, older)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return httpMatchURL, c.Create(ctx, httpRoute)
		}
		return "", err
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		older.Spec = httpRoute.Spec
		return c.Update(ctx, older)
	})
	if retryErr != nil {
		return "", fmt.Errorf("failed to update httproute: %w", retryErr)
	}

	return httpMatchURL, nil
}

func (c *Controller) ensureFinalizer(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) error {
	if ctrlutil.AddFinalizer(flow, KantaloupeControllerFinalizer) {
		return c.Client.Update(ctx, flow)
	}
	return nil
}

func (c *Controller) removeFinalizer(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) error {
	if ctrlutil.RemoveFinalizer(flow, KantaloupeControllerFinalizer) {
		return c.Client.Update(ctx, flow)
	}
	return nil
}

func (c *Controller) createAptResourcesAndPipConfig(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) error {
	if !isKantaloupeflowEabledPlugin(flow) {
		return nil
	}

	currentNamespace := namespace.GetCurrentNamespaceOrDefault()
	ownerReference := []metav1.OwnerReference{
		{
			APIVersion: flow.APIVersion,
			Kind:       flow.Kind,
			Name:       flow.Name,
			UID:        flow.GetUID(),
			Controller: ptr.To(true),
		},
	}

	// create or update the apt resource configmap.
	aptResource := &corev1.ConfigMap{}
	err := c.Get(ctx, client.ObjectKey{Namespace: currentNamespace, Name: "kantaloupe-apt-resources"}, aptResource)
	if err != nil {
		klog.ErrorS(err, "faild to find apt resources for kantaloupe", "kantaloupe", klog.KObj(flow))
	}

	newAptResource := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s-apt", flow.GetName()),
			Namespace:       flow.GetNamespace(),
			OwnerReferences: ownerReference,
		},
		Data: aptResource.Data,
	}
	if err := c.Create(ctx, newAptResource); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err := c.Update(ctx, newAptResource); err != nil {
				klog.ErrorS(err, "failed to update apt resources configmap for kantaloupe", "kantaloupe", klog.KObj(flow))
				return err
			}
		}
	}

	// create or update the apt resource configmap.
	pipConfig := &corev1.ConfigMap{}
	err = c.Get(ctx, client.ObjectKey{Namespace: currentNamespace, Name: "kantaloupe-pip-conf"}, pipConfig)
	if err != nil {
		klog.ErrorS(err, "faild to find pip config for kantaloupe", "kantaloupe", klog.KObj(flow))
	}

	newPipConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s-pip", flow.GetName()),
			Namespace:       flow.GetNamespace(),
			OwnerReferences: ownerReference,
		},
		Data: pipConfig.Data,
	}
	if err := c.Create(ctx, newPipConfig); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err := c.Update(ctx, newPipConfig); err != nil {
				klog.ErrorS(err, "failed to update pip config configmap for kantaloupe", "kantaloupe", klog.KObj(flow))
				return err
			}
		}
	}

	return nil
}

func (c *Controller) ensureDeployment(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flow.GetName(),
			Namespace: flow.GetNamespace(),
			Labels: labels.Merge(flow.Labels, labels.Set{
				constants.KantaloupeFlowAppLabelKey: flow.GetName(),
			}),
			Annotations: flow.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: flow.APIVersion,
					Kind:       flow.Kind,
					Name:       flow.GetName(),
					UID:        flow.GetUID(),
					Controller: ptr.To(true),
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: flow.Spec.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				constants.KantaloupeFlowAppLabelKey: flow.GetName(),
			}},
			Template: *mutateDeploymentPodTemplate(flow),
			Paused:   flow.Spec.Paused,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType, // TODO: use the real strategy
			},
		},
	}

	old := &appsv1.Deployment{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: deploy.Namespace, Name: deploy.Name}, old); err != nil {
		if apierrors.IsNotFound(err) {
			return c.Create(ctx, deploy)
		}
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := c.Get(ctx, client.ObjectKey{Namespace: deploy.Namespace, Name: deploy.Name}, old)
		if err != nil {
			return err
		}
		old.Spec = deploy.Spec
		return c.Update(ctx, old)
	})
}

func mutateDeploymentPodTemplate(flow *kfv1alpha1.KantaloupeFlow) *corev1.PodTemplateSpec {
	template := flow.Spec.Template.DeepCopy()

	// add labels for service to select this pod.
	if template.Labels == nil {
		template.Labels = map[string]string{}
	}
	template.Labels[constants.KantaloupeFlowAppLabelKey] = flow.GetName()

	if !isKantaloupeflowEabledPlugin(flow) {
		return template
	}

	container := template.Spec.Containers[0]
	template.Spec.Containers[0] = completeContainer(container, flow)
	template.Spec.InitContainers = generateInitContainers()
	template.Spec.Volumes = append(template.Spec.Volumes, generateInitVolumes(flow)...)
	for _, plugin := range flow.Spec.Plugins {
		switch plugin {
		case kfv1alpha1.JupyterPluginType:
			template.Spec.Containers[0].Env = append(template.Spec.Containers[0].Env, corev1.EnvVar{Name: "ENABLE_JYPTER"})
		case kfv1alpha1.SSHPluginType:
			template.Spec.Containers[0].Env = append(template.Spec.Containers[0].Env, corev1.EnvVar{Name: "ENABLE_SSH"})
		case kfv1alpha1.VscodePluginType:
			template.Spec.Containers[0].Env = append(template.Spec.Containers[0].Env, corev1.EnvVar{Name: "ENABLE_VSCODE"})
		}
	}

	return template
}

func generateInitVolumes(flow *kfv1alpha1.KantaloupeFlow) []corev1.Volume {
	volume := []corev1.Volume{
		{
			Name: "shared-data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "pip-config-volume",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "apt-sources",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "custom-pip-config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-pip", flow.GetName()),
					},
				},
			},
		},
		{
			Name: "custom-apt-sources",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-apt", flow.GetName()),
					},
				},
			},
		},
	}
	return volume
}

func completeContainer(container corev1.Container, flow *kfv1alpha1.KantaloupeFlow) corev1.Container {
	// 1. init volumeMount
	container.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "shared-data",
			MountPath: "/usr/local/builtin-script/copy",
		},
		{
			Name:      "pip-config-volume",
			MountPath: "/root/.pip/pip.conf",
			SubPath:   "pip.conf",
		},
		{
			Name:      "apt-sources",
			MountPath: "/etc/apt/sources.list",
			SubPath:   "sources.list",
		},
	}

	// 2. init readinessProbe if ssh plugin exists
	if slices.Contains(flow.Spec.Plugins, kfv1alpha1.SSHPluginType) {
		container.ReadinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.IntOrString{
						IntVal: 22,
					},
				},
			},
			InitialDelaySeconds: 1,
			PeriodSeconds:       3,
			SuccessThreshold:    1,
			FailureThreshold:    240,
		}
	}

	// 3. init livenessProbe
	container.LivenessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/bin/sh",
					"-c",
					"/usr/local/builtin-script/copy/check_s6.sh",
				},
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
	}

	// 4. init lifecycle
	container.Lifecycle = &corev1.Lifecycle{
		PostStart: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/bin/sh",
					"-c",
					"/usr/local/builtin-script/copy/set_s6.sh > /usr/local/builtin-script/copy/postStart.log 2>&1",
				},
			},
		},
	}

	// 5. add jupyter env.
	if slices.Contains(flow.Spec.Plugins, "jupyter") {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  "JUPYTER_BASE_URL",
			Value: getJupeterBaseURL(flow),
		})
	}

	return container
}

func generateInitContainers() []corev1.Container {
	return []corev1.Container{
		{
			Name:  "init-container",
			Image: env.InitImageEnvName.Get(),
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("200m"),
					"memory": resource.MustParse("200Mi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("200m"),
					"memory": resource.MustParse("200Mi"),
				},
			},
			Command: []string{
				"/bin/sh",
				"-c",
				"/app/init_image.sh",
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "shared-data",
					MountPath: "/copy",
				},
				{
					Name:      "pip-config-volume",
					MountPath: "/pip-config-volume",
				},
				{
					Name:      "apt-sources",
					MountPath: "/apt-resources",
				},
				{
					Name:      "custom-pip-config-volume",
					MountPath: "/custom-pip-config-volume",
				},
				{
					Name:      "custom-apt-sources",
					MountPath: "/custom-apt-resources",
				},
			},
		},
	}
}

func (c *Controller) ensureService(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flow.GetName(),
			Namespace: flow.GetNamespace(),
			Labels: labels.Merge(flow.GetLabels(), labels.Set{
				constants.KantaloupeFlowAppLabelKey: flow.GetName(),
			}),
			Annotations: flow.GetAnnotations(),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: flow.APIVersion,
					Kind:       flow.Kind,
					Name:       flow.Name,
					UID:        flow.GetUID(),
					Controller: ptr.To(true),
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				constants.KantaloupeFlowAppLabelKey: flow.GetName(),
			},
		},
	}

	// add port to service.
	ports := []corev1.ServicePort{}
	for _, port := range flow.Spec.Networking {
		ports = append(ports, corev1.ServicePort{
			Name:     port.Name,
			Protocol: corev1.ProtocolTCP,
			Port:     port.Port,
			TargetPort: intstr.IntOrString{
				IntVal: port.Port,
			},
		})
	}
	service.Spec.Ports = ports

	old := &corev1.Service{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(service), old); err != nil {
		if apierrors.IsNotFound(err) {
			return service, c.Create(ctx, service)
		}
		return nil, err
	}

	if !equality.Semantic.DeepEqual(old.Spec, service.Spec) {
		return service, c.Update(ctx, service)
	}
	return service, nil
}

// GetMasterNodeIP could find the one master node IP.
func (c *Controller) GetMasterNodeIP(ctx context.Context) (string, error) {
	nodes := &corev1.NodeList{}
	if err := c.List(ctx, nodes); err != nil {
		return "", err
	}

	sort.Slice(nodes.Items, func(i, j int) bool {
		return nodes.Items[i].CreationTimestamp.Before(&nodes.Items[j].CreationTimestamp)
	})

	var internalIP string
	for _, addr := range nodes.Items[0].Status.Addresses {
		// Using External IP as first priority
		if addr.Type == corev1.NodeExternalIP {
			return addr.Address, nil
		}
		if addr.Type == corev1.NodeInternalIP {
			internalIP = addr.Address
		}
	}
	if len(internalIP) != 0 {
		return internalIP, nil
	}
	return "", nil
}

func (c *Controller) ensureAnnotation(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) error {
	_, ok := flow.Annotations[PodAllocationAnnotation]
	if !ok {
		c.InitAnnotation(ctx, flow)
		return nil
	}
	needupdate, want, err := ifNeedUpdateAnnotation(flow.Annotations[PodAllocationAnnotation])
	if err != nil {
		return err
	}
	if !needupdate {
		return nil
	}
	pods, err := c.getKantaloupeflowPods(ctx, flow)
	if err != nil {
		return err
	}
	if len(pods.Items) != 1 {
		klog.ErrorS(nil, "not have exact one pod", "kantaloupeflow", klog.KObj(flow))
		return nil
	}
	if len(pods.Items[0].Annotations) == 0 {
		return nil
	}
	annotation, exists := pods.Items[0].Annotations[annotations.PodGPUMemoryAnnotation]
	if !exists {
		klog.ErrorS(nil, "no annotation", "pod", klog.KObj(&pods.Items[0]))
		return nil
	}

	allocations, err := annotations.MarshalGPUAllocationAnnotation(annotation)
	if err != nil {
		return err
	}
	wantInt, err := strconv.ParseInt(want, 10, 64)
	if err != nil {
		return err
	}
	for i := range allocations {
		allocations[i].Memory = wantInt
	}

	// remove the oom annotation.
	delete(pods.Items[0].Annotations, annotations.OOMExpansionAnnotation)
	pods.Items[0].Annotations[annotations.PodGPUMemoryAnnotation] = annotations.UnmarshalGPUAllocationAnnotation(allocations)

	err = c.Update(ctx, &pods.Items[0])
	if err != nil {
		return err
	}
	return c.UpdateAnnotationHistory(ctx, flow)
}

func (c *Controller) getKantaloupeflowPods(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) (*corev1.PodList, error) {
	pods := &corev1.PodList{}

	options := client.MatchingLabels(map[string]string{
		constants.KantaloupeFlowAppLabelKey: flow.GetName(),
	})
	err := c.List(ctx, pods, options)
	if err != nil {
		return nil, err
	}
	return pods, nil
}

// check if the reconcile caused by annotation change.
func ifNeedUpdateAnnotation(annotation string) (bool, string, error) {
	values := strings.Split(annotation, ",")
	if len(values) != 2 {
		return false, "", fmt.Errorf("annotation %s has unexpected format", annotation)
	}
	// false means no change and dont need to update.
	if values[0] == values[1] {
		return false, "", nil
	}
	return true, values[0], nil
}

func (c *Controller) UpdateAnnotationHistory(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) error {
	annotation := flow.Annotations[PodAllocationAnnotation]
	values := strings.Split(annotation, ",")
	if len(values) != 2 {
		return fmt.Errorf("annotation %s has unexpected format", annotation)
	}
	values[1] = values[0]
	flow.Annotations[PodAllocationAnnotation] = strings.Join(values, ",")
	return c.Update(ctx, flow)
}

func (c *Controller) InitAnnotation(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) error {
	var limit string
	if flow.Spec.Template.Spec.Containers[0].Resources.Limits == nil {
		return fmt.Errorf("kantaloupeflow %s/%s dont have resource limits", flow.GetNamespace(), flow.GetName())
	}
	limits := flow.Spec.Template.Spec.Containers[0].Resources.Limits
	if v, ok := limits[corev1.ResourceName(constants.NvidiaGPUMemory)]; ok {
		limit = fmt.Sprintf("%d", v.Value())
	}
	if v, ok := limits[corev1.ResourceName(constants.MetaxGPUMemory)]; ok {
		limit = fmt.Sprintf("%d", v.Value())
	}
	annotation := fmt.Sprintf("%s,%s", limit, limit)
	if flow.Annotations == nil {
		flow.Annotations = make(map[string]string)
	}
	flow.Annotations[PodAllocationAnnotation] = annotation
	return c.Update(ctx, flow)
}

// SetupWithManager creates a controller and register to controller manager.
func (c *Controller) SetupWithManager(mgr controllerruntime.Manager) error {
	deploymentPredicateFunc := predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			newer := updateEvent.ObjectNew.(*appsv1.Deployment)
			older := updateEvent.ObjectOld.(*appsv1.Deployment)
			return !equality.Semantic.DeepEqual(newer.Spec, older.Spec)
		},
		DeleteFunc: func(_ event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}

	return controllerruntime.NewControllerManagedBy(mgr).
		For(&kfv1alpha1.KantaloupeFlow{}).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(deploymentPredicateFunc)).
		Named(fmt.Sprintf(ControllerName, c.Cluster)).
		Complete(c)
}

func isKantaloupeflowEabledPlugin(flow *kfv1alpha1.KantaloupeFlow) bool {
	return len(flow.Spec.Plugins) > 0
}

func generateGatewaySectionName(network *kfv1alpha1.Networking, port int) string {
	return fmt.Sprintf("%s--%d", network.Name, port)
}

func generateHTTPRouteMatchURL(network *kfv1alpha1.Networking, flow *kfv1alpha1.KantaloupeFlow) string {
	if network.URL != "" {
		return network.URL
	}
	return env.GatewayEnvBaseURL.Get() + strings.Join([]string{
		flow.Namespace,
		flow.Name,
		network.Name,
	}, "/")
}

func getJupeterBaseURL(flow *kfv1alpha1.KantaloupeFlow) string {
	for _, network := range flow.Spec.Networking {
		if network.Name == "jupyter" {
			return generateHTTPRouteMatchURL(&network, flow)
		}
	}
	return ""
}

// Pod type workflow, the current implementation is not yet mature, use with caution.
func (c *Controller) ensurePod(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flow.GetName(),
			Namespace: flow.GetNamespace(),
			Labels: labels.Merge(flow.Labels, labels.Set{
				constants.KantaloupeFlowAppLabelKey: flow.GetName(),
			}),
			Annotations: flow.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: flow.APIVersion,
					Kind:       flow.Kind,
					Name:       flow.GetName(),
					UID:        flow.GetUID(),
					Controller: ptr.To(true),
				},
			},
		},
		Spec: flow.Spec.Template.Spec,
	}
	if isKantaloupeflowEabledPlugin(flow) {
		pod.Spec.InitContainers = generateInitContainers()
		pod.Spec.Volumes = append(pod.Spec.Volumes, generateInitVolumes(flow)...)
		container := pod.Spec.Containers[0]

		if slices.Contains(flow.Spec.Plugins, "jupyter") {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  "JUPYTER_BASE_URL",
				Value: getJupeterBaseURL(flow),
			})
		}

		pod.Spec.Containers[0] = completeContainer(container, flow)
	}

	olds := &corev1.PodList{}
	if err := c.Client.List(ctx, olds, client.InNamespace(flow.Namespace), client.MatchingLabels{constants.KantaloupeFlowAppLabelKey: flow.Name}); err != nil {
		return nil, err
	}

	if len(olds.Items) == 0 {
		// Using the gpu memory annotation instead of resource limit.
		if flow.Annotations != nil {
			if _, ok := flow.Annotations[annotations.PodGPUMemoryAnnotation]; ok {
				values := strings.Split(flow.Annotations[annotations.PodGPUMemoryAnnotation], ",")
				if pod.Spec.Containers[0].Resources.Limits != nil {
					if _, ok := pod.Spec.Containers[0].Resources.Limits[corev1.ResourceName(constants.NvidiaGPUMemory)]; ok {
						pod.Spec.Containers[0].Resources.Limits[corev1.ResourceName(constants.NvidiaGPUMemory)] = resource.MustParse(values[0])
					} else if _, ok := pod.Spec.Containers[0].Resources.Limits[corev1.ResourceName(constants.MetaxGPUMemory)]; ok {
						pod.Spec.Containers[0].Resources.Limits[corev1.ResourceName(constants.MetaxGPUMemory)] = resource.MustParse(values[0])
					}
				}
			}
		}
		return pod, c.Client.Create(ctx, pod)
	}

	old := olds.Items[0]
	if !utils.MapsEqual(old.Spec.Containers[0].Resources.Limits, pod.Spec.Containers[0].Resources.Limits) || old.Spec.Containers[0].Image != pod.Spec.Containers[0].Image {
		old.Spec.Containers[0].Resources.Limits = pod.Spec.Containers[0].Resources.Limits
		old.Spec.Containers[0].Image = pod.Spec.Containers[0].Image
		return pod, c.Client.Update(ctx, &old)
	}
	return &old, nil
}

func (c *Controller) syncKantaloupeFlowStatus(ctx context.Context, flow *kfv1alpha1.KantaloupeFlow) error {
	pods := &corev1.PodList{}
	if err := c.Client.List(ctx, pods, client.InNamespace(flow.Namespace), client.MatchingLabels{constants.KantaloupeFlowAppLabelKey: flow.Name}); err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return nil
	}
	Conditions := convertPodCondition(pods.Items[0].Status.Conditions)
	if equality.Semantic.DeepEqual(flow.Status.Conditions, Conditions) {
		return nil
	}
	flow.Status.Conditions = Conditions
	return c.Status().Update(ctx, flow)
}

func convertPodCondition(conditions []corev1.PodCondition) []metav1.Condition {
	res := []metav1.Condition{}
	for _, cond := range conditions {
		res = append(res, metav1.Condition{
			Type:               string(cond.Type),
			Status:             metav1.ConditionStatus(cond.Status),
			Reason:             cond.Reason,
			Message:            cond.Message,
			LastTransitionTime: cond.LastTransitionTime,
		})
	}
	return res
}
