package kantaloupeflow

import (
	"context"
	"slices"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	flowcrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/env"
)

const (
	nameRandomLength        = 5
	passwordRandomLength    = 15
	systemNetworkNamePrefix = "system-"
	customNetworkNamePrefix = "custom-"
)

type Service interface {
	CreateKantaloupeflow(ctx context.Context, cluster string, flow *flowcrdv1alpha1.KantaloupeFlow) (*flowcrdv1alpha1.KantaloupeFlow, error)
	GetKantaloupeflow(ctx context.Context, cluster, namespace, name string) (*flowcrdv1alpha1.KantaloupeFlow, error)
	DeleteKantaloupeflow(ctx context.Context, cluster, namespace, name string) error
	ListKantaloupeflows(ctx context.Context, cluster, namespace string) ([]*flowcrdv1alpha1.KantaloupeFlow, error)
	UpdataKantaloupeflow(ctx context.Context, cluster string, flow *flowcrdv1alpha1.KantaloupeFlow) error
}

type service struct {
	clientManager engine.ClientManagerInterface
}

func NewService(clientManager engine.ClientManagerInterface) Service {
	return &service{
		clientManager: clientManager,
	}
}

func (s *service) CreateKantaloupeflow(ctx context.Context, cluster string, flow *flowcrdv1alpha1.KantaloupeFlow) (*flowcrdv1alpha1.KantaloupeFlow, error) {
	c, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	// create networkings for kantaloupeflow.
	networkings := []flowcrdv1alpha1.Networking{}
	container := flow.Spec.Template.Spec.Containers[0]
	// TODO: only apply for nvidia gpu.
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  constants.EnvLibCudaLogLevel,
		Value: env.EnvLibCudaLogLevel.Get(),
	})
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "GPU_CORE_UTILIZATION_POLICY",
		Value: "force",
	})

	if slices.Contains(flow.Spec.Plugins, flowcrdv1alpha1.SSHPluginType) {
		networkings = append(networkings, flowcrdv1alpha1.Networking{
			Name:     constants.SSHServiceName,
			Type:     constants.NetworkTCPRoute,
			Protocol: constants.TCPProtocol,
			Port:     constants.DefaultPortSSH,
		})
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  constants.EnvSSHRootPasswordKey,
			Value: utils.RandomString(passwordRandomLength),
		})
	}
	if slices.Contains(flow.Spec.Plugins, flowcrdv1alpha1.VscodePluginType) {
		networkings = append(networkings, flowcrdv1alpha1.Networking{
			Name:     constants.VSCodeServiceName,
			Type:     constants.NetworkHTTPRoute,
			Protocol: constants.TCPProtocol,
			Port:     constants.DefaultPortVSCode,
		})
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  constants.EnvEnableVSCode,
			Value: "true",
		})
	}
	if slices.Contains(flow.Spec.Plugins, flowcrdv1alpha1.JupyterPluginType) {
		networkings = append(networkings, flowcrdv1alpha1.Networking{
			Name:     constants.JupyterServiceName,
			Type:     constants.NetworkHTTPRoute,
			Protocol: constants.HTTPProtocol,
			Port:     constants.DefaultPortJupyter,
		})
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  constants.EnvEnableJupyter,
			Value: "true",
		})
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  constants.EnvJupyterToken,
			Value: rand.String(64),
		})
	}

	flow.Spec.Networking = networkings
	flow.Spec.Template.Spec.Containers[0] = container

	if err := c.Create(ctx, flow); err != nil {
		klog.ErrorS(err, "failed to create kantaloupeflow", "kantaloupeflow", klog.KObj(flow))
		return nil, err
	}

	return flow, nil
}

func (s *service) GetKantaloupeflow(ctx context.Context, cluster, namespace, name string) (*flowcrdv1alpha1.KantaloupeFlow, error) {
	c, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	flow := &flowcrdv1alpha1.KantaloupeFlow{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, flow); err != nil {
		klog.ErrorS(err, "failed to get kantaloupeflow", "kantaloupeflow", klog.KObj(flow))
		return nil, err
	}

	return flow, nil
}

func (s *service) DeleteKantaloupeflow(ctx context.Context, cluster, namespace, name string) error {
	c, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return err
	}
	return c.Delete(ctx, &flowcrdv1alpha1.KantaloupeFlow{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
}

func (s *service) ListKantaloupeflows(ctx context.Context, cluster, namespace string) ([]*flowcrdv1alpha1.KantaloupeFlow, error) {
	c, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return nil, err
	}

	if namespace == constants.SelectAll {
		namespace = corev1.NamespaceAll
	}

	flows := &flowcrdv1alpha1.KantaloupeFlowList{}
	if err := c.List(ctx, flows, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	return utils.SliceToPointerSlice(flows.Items), nil
}

func (s *service) UpdataKantaloupeflow(ctx context.Context, cluster string, flow *flowcrdv1alpha1.KantaloupeFlow) error {
	c, err := s.clientManager.GeteClient(cluster)
	if err != nil {
		return err
	}

	if err := c.Update(ctx, flow); err != nil {
		klog.ErrorS(err, "update flow error", "kantaloupeflow", klog.KObj(flow))
		return err
	}
	return nil
}
