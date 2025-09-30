package bff

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/klog/v2"

	clustersv1alpha1 "github.com/dynamia-ai/kantaloupe/api/clusters/v1alpha1"
	flowcrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1"
	flowv1alpha1 "github.com/dynamia-ai/kantaloupe/api/kantaloupeflow/v1alpha1"
	kantaloupeapi "github.com/dynamia-ai/kantaloupe/api/v1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/service/cluster"
	"github.com/dynamia-ai/kantaloupe/pkg/service/core"
	kfservice "github.com/dynamia-ai/kantaloupe/pkg/service/kantaloupeflow"
	"github.com/dynamia-ai/kantaloupe/pkg/service/monitoring"
	"github.com/dynamia-ai/kantaloupe/pkg/service/quota"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/annotations"
)

var _ kantaloupeapi.KantaloupeflowServer = &KantaloupeflowHandler{}

const PodAllocationAnnotation = "kantaloupe.dynamia.io/pod-allocation-meet"

type KantaloupeflowHandler struct {
	kantaloupeapi.UnimplementedKantaloupeflowServer
	service           kfservice.Service
	workloadService   core.Service
	clusterService    cluster.Service
	quotaService      quota.Service
	monitoringService monitoring.Service
}

// NewKantaloupeflowHandler new a cluster handler by client manager.
func NewKantaloupeflowHandler(clientManager engine.ClientManagerInterface, prometheus engine.PrometheusInterface) *KantaloupeflowHandler {
	return &KantaloupeflowHandler{
		service:           kfservice.NewService(clientManager),
		clusterService:    cluster.NewService(clientManager),
		workloadService:   core.NewService(clientManager),
		quotaService:      quota.NewService(clientManager, prometheus),
		monitoringService: monitoring.NewService(prometheus),
	}
}

// CreateKaantaloupeflow create a kantaloupeflow.
func (h *KantaloupeflowHandler) CreateKantaloupeflow(ctx context.Context, req *flowv1alpha1.CreateKantaloupeflowRequest) (*flowv1alpha1.Kantaloupeflow, error) {
	if errs := validation.IsDNS1035Label(req.GetData().GetMetadata().GetName()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow name %s is invalid, error: %s", req.GetData().GetMetadata().GetName(), errs)
	}
	if !utils.IsValidLabelNames(req.GetData().GetMetadata().GetLabels()) {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow label %s is invalid", req.GetData().GetMetadata().GetLabels())
	}
	if !utils.IsValidAnnotationNames(req.GetData().GetMetadata().GetAnnotations()) {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow annotation %s is invalid", req.GetData().GetMetadata().GetAnnotations())
	}
	if req.GetData().GetSpec() == nil || req.GetData().GetSpec().GetTemplate() == nil || req.GetData().GetSpec().GetTemplate().GetSpec() == nil ||
		len(req.GetData().GetSpec().GetTemplate().GetSpec().GetContainers()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "podTemplate can not be emplty")
	}

	// TODO: Frontend dont have this paremeter yet. Default using deployment.
	req.Data.Spec.Workload = flowv1alpha1.WorkloadType_Deployment

	cluster, err := h.clusterService.GetCluster(ctx, req.GetCluster())
	if err != nil {
		return nil, err
	}
	flow := ConvertProto2Kantaloupeflow(req.Data)
	err = patchByProvider(flow, cluster.Spec.Provider)
	if err != nil {
		return nil, err
	}
	flow, err = h.service.CreateKantaloupeflow(ctx, req.GetCluster(), flow)
	if err != nil {
		return nil, err
	}
	return ConvertKantaloupeflow2Proto(flow), nil
}

// GetKantaloupeflow get a specified Kantaloupeflow.
func (h *KantaloupeflowHandler) GetKantaloupeflow(ctx context.Context, req *flowv1alpha1.GetKantaloupeflowRequest) (*flowv1alpha1.GetKantaloupeflowResponse, error) {
	if errs := validation.IsDNS1035Label(req.GetName()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow name %s is invalid, error: %s", req.Name, errs)
	}
	if errs := validation.IsDNS1035Label(req.Namespace); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow namespace %s is invalid, error: %s", req.Namespace, errs)
	}

	flow, err := h.service.GetKantaloupeflow(ctx, req.Cluster, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}
	res := &flowv1alpha1.GetKantaloupeflowResponse{
		Kantaloupeflow: ConvertKantaloupeflow2Proto(flow),
	}
	res.Kantaloupeflow.Status.Gpus, res.Node, err = h.getKantaloupeflowGpus(ctx, req.GetCluster(), req.GetNamespace(), req.GetName())
	if err != nil {
		return nil, err
	}

	return res, nil
}

// DeleteKantaloupeflow delete a specified Kantaloupeflow.
func (h *KantaloupeflowHandler) DeleteKantaloupeflow(ctx context.Context, req *flowv1alpha1.DeleteKantaloupeflowRequest) (*emptypb.Empty, error) {
	if errs := validation.IsDNS1035Label(req.Cluster); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow cluster %s is invalid, error: %s", req.Cluster, errs)
	}
	if errs := validation.IsDNS1035Label(req.Name); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow name %s is invalid, error: %s", req.Name, errs)
	}
	if errs := validation.IsDNS1035Label(req.Namespace); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow namespace %s is invalid, error: %s", req.Namespace, errs)
	}

	return &emptypb.Empty{}, h.service.DeleteKantaloupeflow(ctx, req.Cluster, req.Namespace, req.Name)
}

// ListKantaloupeflows lists Kantaloupeflow.
func (h *KantaloupeflowHandler) ListKantaloupeflows(ctx context.Context, req *flowv1alpha1.ListKantaloupeflowsRequest) (*flowv1alpha1.ListKantaloupeflowsResponse, error) {
	if req.Name != "" {
		if errs := validation.IsDNS1035Label(req.Name); len(errs) != 0 {
			return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow name %s is invalid, error: %s", req.Name, errs)
		}
	}

	flows, err := h.service.ListKantaloupeflows(ctx, req.Cluster, req.Namespace)
	if err != nil {
		return nil, err
	}

	filtered := filterKantaloupeflow(flows, req.GetName(), req.GetStatus())
	// TODO: use utils.SortStructSlice
	sortByMetaFields(filtered, req.GetSortBy().String(), req.GetSortDir().String())
	paged := utils.PagedItems(filtered, req.GetPage(), req.GetPageSize())

	items := ConvertKantaloupeflows2Proto(paged)
	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)

	// TODO: optimize
	for i := range items {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			gpus, _, err := h.getKantaloupeflowGpus(ctx, req.GetCluster(), items[i].Metadata.Namespace, items[i].Metadata.Name)
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}
			items[i].Status.Gpus = gpus
		}(i)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	return &flowv1alpha1.ListKantaloupeflowsResponse{
		Items:      items,
		Pagination: utils.NewPage(req.Page, req.PageSize, len(filtered)),
	}, nil
}

// TODO: optimzie the function.
func sortByMetaFields(list []*flowcrdv1alpha1.KantaloupeFlow, field, asc string) {
	sort.Slice(list, func(i, j int) bool {
		var less bool

		switch field {
		case "field_name":
			less = list[i].GetName() < list[j].GetName()
		case "created_at":
			less = !list[i].GetCreationTimestamp().After(list[j].GetCreationTimestamp().Time)
		default:
			less = !list[i].GetCreationTimestamp().After(list[j].GetCreationTimestamp().Time)
		}

		if asc == constants.SortByDesc {
			return !less
		}
		return less
	})
}

func (h *KantaloupeflowHandler) GetKantaloupeTree(ctx context.Context, _ *emptypb.Empty) (*flowv1alpha1.KantaloupeTree, error) {
	clusters, err := h.clusterService.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	data := []*flowv1alpha1.KantaloupeTreeNode{}
	for _, cluster := range clusters {
		if convertCondition2State(cluster.Status) != clustersv1alpha1.ClusterState_RUNNING {
			continue
		}
		labelKey, err := labels.NewRequirement(constants.KantaloupeFlowAppLabelKey, selection.Exists, nil)
		if err != nil {
			return nil, err
		}
		labelsSelector := labels.NewSelector().Add(*labelKey)

		pods, err := h.workloadService.ListPods(ctx, cluster.GetName(), "", labelsSelector.String())
		if err != nil {
			return nil, err
		}

		clusterTreeNode := &flowv1alpha1.KantaloupeTreeNode{
			Name:     cluster.GetName(),
			Value:    int32(len(pods)),
			Children: []*flowv1alpha1.KantaloupeTreeNode{},
		}

		tmp := map[string]int32{}
		for _, pod := range pods {
			if node := pod.Spec.NodeName; len(node) > 0 {
				tmp[node]++
			}
		}
		for node, num := range tmp {
			clusterTreeNode.Children = append(clusterTreeNode.Children, &flowv1alpha1.KantaloupeTreeNode{
				Name:  node,
				Value: num,
			})
		}
		data = append(data, clusterTreeNode)
	}

	return &flowv1alpha1.KantaloupeTree{Data: data}, nil
}

func (h *KantaloupeflowHandler) UpdateKantaloupeflowGPUMemory(ctx context.Context, req *flowv1alpha1.UpdateKantaloupeflowGPUMemoryRequest) (*emptypb.Empty, error) {
	if errs := validation.IsDNS1035Label(req.GetName()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow name %s is invalid, error: %s", req.Name, errs)
	}
	if errs := validation.IsDNS1035Label(req.Namespace); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow namespace %s is invalid, error: %s", req.Namespace, errs)
	}
	if req.Gpumemory <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow memory %d is invalid", req.Gpumemory)
	}

	flow, err := h.service.GetKantaloupeflow(ctx, req.Cluster, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	if flow.Annotations == nil {
		flow.Annotations = map[string]string{}
	}

	values := strings.Split(flow.Annotations[PodAllocationAnnotation], ",")
	if len(values) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow annotation %s is invalid", PodAllocationAnnotation)
	}

	// Get quota for the namespace.
	quotas, err := h.quotaService.ListQuotas(ctx, req.Namespace, req.Cluster)
	if err != nil {
		return nil, err
	}

	// Check nvidia resource.
	for _, quota := range quotas {
		if _, ok := quota.Status.Hard[constants.NvidiaQuotaGpuMemory]; !ok {
			continue
		}
		limit := quota.Status.Hard[constants.NvidiaQuotaGpuMemory]
		used := quota.Status.Used[constants.NvidiaQuotaGpuMemory]

		using, err := strconv.Atoi(values[0])
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow annotation %s is invalid", PodAllocationAnnotation)
		}
		// Check the limit of gpumemory
		if used.Value()-int64(using)+int64(req.Gpumemory) > limit.Value() {
			return nil, status.Errorf(codes.ResourceExhausted, "the quota group has no enough gpumemory resource")
		}
	}

	values[1] = values[0]
	values[0] = fmt.Sprintf("%d", req.Gpumemory)
	flow.Annotations[PodAllocationAnnotation] = strings.Join(values, ",")

	err = h.service.UpdataKantaloupeflow(ctx, req.Cluster, flow)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func filterKantaloupeflow(list []*flowcrdv1alpha1.KantaloupeFlow, keyword string, state flowv1alpha1.KantaloupeflowState) []*flowcrdv1alpha1.KantaloupeFlow {
	res := []*flowcrdv1alpha1.KantaloupeFlow{}
	for _, flow := range list {
		if !utils.MatchByFuzzyName(flow, keyword) {
			continue
		}

		if state == flowv1alpha1.KantaloupeflowState_KANTALOUPEFLOW_STATE_UNSPECIFIED {
			res = append(res, flow)
			continue
		}

		flowState := calculateKantaloupeflowState(flow.Status.Conditions)
		if flowState == state {
			res = append(res, flow)
		}
	}
	return res
}

func (h *KantaloupeflowHandler) getKantaloupeflowGpus(ctx context.Context, clusterName, namespace, flowName string) ([]*flowv1alpha1.GPU, string, error) {
	labelSelector := labels.Set{constants.KantaloupeFlowAppLabelKey: flowName}
	pods, err := h.workloadService.ListPods(ctx, clusterName, namespace, labelSelector.String())
	if err != nil {
		return nil, "", err
	}
	if len(pods) == 0 {
		return nil, "", nil
	}

	pod := pods[0]
	cluster, err := h.clusterService.GetCluster(ctx, clusterName)
	if err != nil {
		return nil, "", err
	}

	if cluster.Spec.Type == clustersv1alpha1.ClusterType_METAX.String() {
		gpus, err := h.getMetaxGpus(ctx, pod, clusterName, namespace, flowName)
		if err != nil {
			klog.ErrorS(err, "failed to get metax gpus")
		}
		return gpus, pod.Spec.NodeName, nil
	}
	if cluster.Spec.Type == clustersv1alpha1.ClusterType_NVIDIA.String() {
		gpus, err := h.getNvidiaGpus(ctx, clusterName, pod)
		if err != nil {
			klog.ErrorS(err, "failed to get nvidia gpus")
		}
		return gpus, pod.Spec.NodeName, nil
	}
	if cluster.Spec.Type == clustersv1alpha1.ClusterType_ASCEND.String() {
		gpus, err := h.getAscendGpus(pod)
		if err != nil {
			klog.ErrorS(err, "failed to get ascend gpus")
		}
		return gpus, pod.Spec.NodeName, nil
	}
	if cluster.Spec.Type == clustersv1alpha1.ClusterType_NEURON.String() {
		gpus, err := h.getNeuronGpus(pod)
		if err != nil {
			klog.ErrorS(err, "failed to get neuron gpus")
		}
		return gpus, pod.Spec.NodeName, nil
	}

	return nil, "", nil
}

func (h *KantaloupeflowHandler) getMetaxGpus(ctx context.Context, pod *corev1.Pod, clusterName, namespace, flowName string) ([]*flowv1alpha1.GPU, error) {
	gpuList := []*flowv1alpha1.GPU{}

	var vmemory int64
	var vcore float64

	if memory, ok := pod.Spec.Containers[0].Resources.Limits["metax-tech.com/vmemory"]; ok {
		if val, ok := memory.AsInt64(); ok {
			vmemory = val
		}
	}
	if core, ok := pod.Spec.Containers[0].Resources.Limits["metax-tech.com/vcore"]; ok {
		vcore = core.AsFloat64Slow()
	}

	vec, err := h.monitoringService.QueryWorkload(ctx, clusterName, namespace, flowName, monitoring.QueryTypeGPUMemoryUsed)
	if err != nil {
		return gpuList, err
	}

	gpuList = append(gpuList, &flowv1alpha1.GPU{
		Uuid:   string(vec[0].Metric["UUID"]),
		Model:  string(vec[0].Metric["modelName"]),
		Memory: int32(vmemory),
		Core:   float32(vcore),
	})
	return gpuList, nil
}

func (h *KantaloupeflowHandler) getNvidiaGpus(ctx context.Context, clusterName string, pod *corev1.Pod) ([]*flowv1alpha1.GPU, error) {
	gpuList := []*flowv1alpha1.GPU{}
	if annotation, exists := pod.Annotations[annotations.PodGPUMemoryAnnotation]; exists {
		allocations, err := annotations.MarshalGPUAllocationAnnotation(annotation)
		if err != nil {
			return nil, err
		}
		for i := range allocations {
			gpuList = append(gpuList, &flowv1alpha1.GPU{
				Uuid:   allocations[i].UUID,
				Memory: int32(allocations[i].Memory),
				Core:   float32(allocations[i].Core) / 100,
			})
		}
	}
	for i := range gpuList {
		vec, err := h.monitoringService.QueryGPU(ctx, clusterName, gpuList[i].Uuid, monitoring.GPUQueryTypeMemoryUsed)
		if err != nil {
			return nil, err
		}
		gpuList[i].Model = string(vec[0].Metric["modelName"])
	}
	return gpuList, nil
}

func (h *KantaloupeflowHandler) getAscendGpus(pod *corev1.Pod) ([]*flowv1alpha1.GPU, error) {
	gpuList := []*flowv1alpha1.GPU{}
	for key, value := range pod.Annotations {
		if !strings.Contains(key, "-devices-allocated") {
			continue
		}
		allocations, err := annotations.MarshalGPUAllocationAnnotation(value)
		if err != nil {
			return nil, err
		}
		for i := range allocations {
			gpuList = append(gpuList, &flowv1alpha1.GPU{
				Uuid:   allocations[i].UUID,
				Model:  allocations[i].Vendor,
				Memory: int32(allocations[i].Memory),
			})
		}
	}
	return gpuList, nil
}

func (h *KantaloupeflowHandler) getNeuronGpus(pod *corev1.Pod) ([]*flowv1alpha1.GPU, error) {
	gpuList := []*flowv1alpha1.GPU{}
	if annotation, exists := pod.Annotations[annotations.PodNeuronsAnnotation]; exists {
		allocations, err := annotations.MarshalGPUAllocationAnnotation(annotation)
		if err != nil {
			return nil, err
		}
		for i := range allocations {
			gpuList = append(gpuList, &flowv1alpha1.GPU{
				Uuid:   allocations[i].UUID,
				Model:  allocations[i].Vendor,
				Memory: int32(allocations[i].Memory),
				Core:   float32(allocations[i].Core),
			})
		}
	}
	return gpuList, nil
}

func (h *KantaloupeflowHandler) GetKantaloupeflowConditions(ctx context.Context, req *flowv1alpha1.GetKantaloupeflowConditionsRequest) (*flowv1alpha1.GetKantaloupeflowConditionsResponse, error) {
	res := []*flowv1alpha1.ConditionStrings{}
	if errs := validation.IsDNS1035Label(req.GetName()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow name %s is invalid, error: %s", req.Name, errs)
	}
	if errs := validation.IsDNS1035Label(req.Namespace); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "kantaloupeflow namespace %s is invalid, error: %s", req.Namespace, errs)
	}
	flow, err := h.service.GetKantaloupeflow(ctx, req.Cluster, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}
	for _, condition := range flow.Status.Conditions {
		res = append(res, &flowv1alpha1.ConditionStrings{
			Type:    condition.Type,
			Status:  string(condition.Status),
			Message: condition.Message,
		})
	}
	event, err := h.workloadService.ListEventsByDeployment(ctx, req.Cluster, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}
	if len(event) > 0 {
		res = append(res, &flowv1alpha1.ConditionStrings{
			Type:    event[0].Type,
			Status:  event[0].Reason,
			Message: event[0].Message,
		})
	}

	return &flowv1alpha1.GetKantaloupeflowConditionsResponse{Conditions: res}, nil
}
