package bff

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	corev1alpha1 "github.com/dynamia-ai/kantaloupe/api/core/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/api/types"
	kantaloupeapi "github.com/dynamia-ai/kantaloupe/api/v1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/service/cluster"
	"github.com/dynamia-ai/kantaloupe/pkg/service/core"
	"github.com/dynamia-ai/kantaloupe/pkg/service/monitoring"
	quotaservice "github.com/dynamia-ai/kantaloupe/pkg/service/quota"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/annotations"
	nodeutil "github.com/dynamia-ai/kantaloupe/pkg/utils/node"
)

var _ kantaloupeapi.CoreServer = &CoreHandler{}

type CoreHandler struct {
	sync.Mutex
	kantaloupeapi.UnimplementedCoreServer
	service           core.Service
	clusterService    cluster.Service
	monitoringService monitoring.Service
	quotaService      quotaservice.Service
}

func NewCoreHandler(clientManager engine.ClientManagerInterface, prometheus engine.PrometheusInterface) *CoreHandler {
	monitoringService := monitoring.NewService(prometheus)
	return &CoreHandler{
		clusterService:    cluster.NewService(clientManager),
		service:           core.NewService(clientManager),
		monitoringService: monitoringService,
		quotaService:      quotaservice.NewService(clientManager, prometheus),
	}
}

func (h *CoreHandler) CreatePersistentVolume(ctx context.Context, req *corev1alpha1.CreatePersistentVolumeRequest) (*corev1alpha1.CreatePersistentVolumeResponse, error) {
	pv, err := convertProto2PersistentVolume(req.GetData())
	if err != nil {
		return nil, err
	}
	if err = validNamespaceAndName(pv); err != nil {
		return nil, err
	}

	pv, err = h.service.CreatePersistentVolume(ctx, req.GetCluster(), pv)
	if err != nil {
		return nil, err
	}
	return &corev1alpha1.CreatePersistentVolumeResponse{Data: convertPersistentVolume2Proto(pv)}, nil
}

func validNamespaceAndName(obj metav1.Object) error {
	if errs := validation.IsValidLabelValue(obj.GetName()); len(errs) > 0 {
		return status.Errorf(codes.InvalidArgument, "Failed to validate resource name %s", strings.Join(errs, ","))
	}

	if obj.GetNamespace() != "" {
		if errs := validation.IsDNS1123Label(obj.GetNamespace()); len(errs) > 0 {
			return status.Errorf(codes.InvalidArgument, "Failed to validate resource namespace %s", strings.Join(errs, ","))
		}
	}
	return nil
}

func (h *CoreHandler) DeletePersistentVolume(ctx context.Context, req *corev1alpha1.DeletePersistentVolumeRequest) (*emptypb.Empty, error) {
	err := h.service.DeletePersistentVolume(ctx, req.GetCluster(), req.GetName())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (h *CoreHandler) GetPersistentVolume(ctx context.Context, req *corev1alpha1.GetPersistentVolumeRequest) (*corev1alpha1.GetPersistentVolumeResponse, error) {
	pv, err := h.service.GetPersistentVolume(ctx, req.GetCluster(), req.GetName())
	if err != nil {
		return nil, err
	}
	return &corev1alpha1.GetPersistentVolumeResponse{Data: convertPersistentVolume2Proto(pv)}, nil
}

func (h *CoreHandler) ListPersistentVolumes(ctx context.Context, req *corev1alpha1.ListPersistentVolumesRequest) (*corev1alpha1.ListPersistentVolumesResponse, error) {
	pvs, err := h.service.ListPersistentVolumes(ctx, req.GetCluster())
	if err != nil {
		return nil, err
	}

	filtered := utils.FilterByFuzzyName(pvs, req.GetName())

	// TODO: add map
	if err = utils.SortStructSlice(filtered, req.GetSortOption().GetField(), req.GetSortOption().GetAsc(), utils.SnakeToCamelMapper()); err != nil {
		return nil, err
	}
	paged := utils.PagedItems(filtered, req.GetPage(), req.GetPageSize())
	items := convertPersistentVolumes2Proto(paged)

	return &corev1alpha1.ListPersistentVolumesResponse{
		Items:      items,
		Pagination: utils.NewPage(req.GetPage(), req.GetPageSize(), len(filtered)),
	}, nil
}

func (h *CoreHandler) UpdatePersistentVolume(ctx context.Context, req *corev1alpha1.UpdatePersistentVolumeRequest) (*corev1alpha1.UpdatePersistentVolumeResponse, error) {
	pv, err := convertProto2PersistentVolume(req.GetData())
	if err != nil {
		return nil, err
	}

	if err = validNamespaceAndName(pv); err != nil {
		return nil, err
	}

	pv, err = h.service.UpdatePersistentVolume(ctx, req.GetCluster(), pv)
	if err != nil {
		return nil, err
	}
	return &corev1alpha1.UpdatePersistentVolumeResponse{Data: convertPersistentVolume2Proto(pv)}, nil
}

func (h *CoreHandler) CreateSecret(ctx context.Context, req *corev1alpha1.CreateSecretRequest) (*corev1alpha1.CreateSecretResponse, error) {
	secret, err := convertProto2Secret(req.GetData())
	if err != nil {
		return nil, err
	}
	err = validNamespaceAndName(secret)
	if err != nil {
		return nil, err
	}
	secret, err = h.service.CreateSecret(ctx, req.GetCluster(), req.GetNamespace(), secret)
	if err != nil {
		return nil, err
	}
	return &corev1alpha1.CreateSecretResponse{Data: convertSecret2Proto(*secret)}, nil
}

func (h *CoreHandler) DeleteSecret(ctx context.Context, req *corev1alpha1.DeleteSecretRequest) (*emptypb.Empty, error) {
	err := h.service.DeleteSecret(ctx, req.GetCluster(), req.GetNamespace(), req.GetName())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (h *CoreHandler) GetSecret(ctx context.Context, req *corev1alpha1.GetSecretRequest) (*corev1alpha1.Secret, error) {
	secret, err := h.service.GetSecret(ctx, req.GetCluster(), req.GetNamespace(), req.GetName())
	if err != nil {
		return nil, err
	}
	return convertSecret2Proto(*secret), nil
}

func (h *CoreHandler) ListSecrets(ctx context.Context, req *corev1alpha1.ListSecretsRequest) (*corev1alpha1.ListSecretsResponse, error) {
	secrets, err := h.service.ListSecrets(ctx, req.GetCluster(), req.GetNamespace())
	if err != nil {
		return nil, err
	}

	filtered := utils.FilterByFuzzyName(secrets, req.GetName())
	// TODO: add map.
	if err = utils.SortStructSlice(filtered, req.GetSortOption().GetField(), req.GetSortOption().GetAsc(), utils.SnakeToCamelMapper()); err != nil {
		return nil, err
	}
	paged := utils.PagedItems(filtered, req.GetPage(), req.GetPageSize())
	items := convertSecrets2Proto(paged)

	return &corev1alpha1.ListSecretsResponse{
		Items:      items,
		Pagination: utils.NewPage(req.GetPage(), req.GetPageSize(), len(filtered)),
	}, nil
}

func (h *CoreHandler) ListClusterNamespaces(ctx context.Context, req *corev1alpha1.ListClusterNamespacesRequest) (*corev1alpha1.ListClusterNamespacesResponse, error) {
	namespaces, err := h.service.ListNamespaces(ctx, req.GetCluster())
	if err != nil {
		return nil, err
	}

	// add resource quota message to namespace.
	resourceQuotaMap := map[string][]string{}
	if req.GetResourceQuota() {
		resourceQuotas, err := h.quotaService.ListQuotas(ctx, metav1.NamespaceAll, req.GetCluster())
		if err != nil {
			return nil, err
		}
		for _, rq := range resourceQuotas {
			if _, ok := rq.Labels[constants.ManagedByLabelKey]; ok {
				resourceQuotaMap[rq.Namespace] = append(resourceQuotaMap[rq.Namespace], rq.Name)
			}
		}
	}

	filtered := utils.FilterByFuzzyName(namespaces, req.GetName())
	// TODO: add map
	if err = utils.SortStructSlice(filtered, req.GetSortOption().GetField(), req.GetSortOption().GetAsc(), utils.SnakeToCamelMapper()); err != nil {
		return nil, err
	}
	paged := utils.PagedItems(filtered, req.GetPage(), req.GetPageSize())

	items := []*corev1alpha1.Namespace{}
	for _, namespace := range paged {
		items = append(items, &corev1alpha1.Namespace{Name: namespace.Name, ResourceQuotas: resourceQuotaMap[namespace.Name]})
	}

	return &corev1alpha1.ListClusterNamespacesResponse{
		Items:      items,
		Pagination: utils.NewPage(req.GetPage(), req.GetPageSize(), len(filtered)),
	}, nil
}

func (h *CoreHandler) ListClusterGPUSummary(ctx context.Context, req *corev1alpha1.ListClusterGPUSummaryRequest) (*corev1alpha1.ListClusterGPUSummaryResponse, error) {
	nodes, err := h.service.ListNodes(ctx, req.GetCluster())
	if err != nil {
		return nil, err
	}

	return listGpuSummaryFrom(nodes), nil
}

// The format of the value of "hami.io/node-mlu-register" refer to the following hami code:
// https://github.com/Project-HAMi/HAMi/blob/master/pkg/util/util.go#L90
func listGpuSummaryFrom(nodes []*corev1.Node) *corev1alpha1.ListClusterGPUSummaryResponse {
	summary := make([]*corev1alpha1.GPUSummary, 0)
	for _, n := range nodes {
		if !nodeutil.IsNodeReady(n) {
			continue
		}

		var vgpuTypes []string
		val, ok := n.Annotations[constants.HamiRegisterAnonationKey]
		if ok {
			gpuInfos := strings.Split(val, ":")
			for _, info := range gpuInfos {
				rets := strings.Split(info, ",")
				if len(rets) == 7 {
					vgpuTypes = append(vgpuTypes, rets[4])
				}
			}

			summary = append(summary, &corev1alpha1.GPUSummary{
				Node:      n.GetName(),
				VgpuTypes: vgpuTypes,
			})
		}
	}
	return &corev1alpha1.ListClusterGPUSummaryResponse{Summary: summary}
}

func (h *CoreHandler) ListClusterEvents(ctx context.Context, req *corev1alpha1.ListClusterEventsRequest) (*corev1alpha1.ListClusterEventsResponse, error) {
	events, err := h.service.ListEvents(ctx, req.GetCluster(), req.GetNamespace())
	if err != nil {
		return nil, err
	}

	// TODO: add map
	if err = utils.SortStructSlice(events, req.GetSortOption().GetField(), req.GetSortOption().GetAsc(), utils.SnakeToCamelMapper()); err != nil {
		return nil, err
	}

	// TODO: why not page?

	items := convertEvents(events)

	return &corev1alpha1.ListClusterEventsResponse{
		Items:      items,
		Pagination: utils.NewPage(req.Page, req.PageSize, len(events)),
	}, nil
}

func (h *CoreHandler) ListEvents(ctx context.Context, req *corev1alpha1.ListEventsRequest) (*corev1alpha1.ListEventsResponse, error) {
	var list []*corev1.Event
	var err error
	switch req.Kind.String() {
	case constants.KindPod:
		list, err = h.service.ListEventsByPod(ctx, req.GetCluster(), req.GetNamespace(), req.GetKindName())
	default:
		list, err = h.service.ListEvents(ctx, req.GetCluster(), req.GetNamespace())
	}
	if err != nil {
		return nil, err
	}

	// TODO: add map
	if err = utils.SortStructSlice(list, req.GetSortOption().GetField(), req.GetSortOption().GetAsc(), utils.SnakeToCamelMapper()); err != nil {
		return nil, err
	}

	// TODO: why not page?

	items := convertEvents(list)
	return &corev1alpha1.ListEventsResponse{
		Items:      items,
		Pagination: utils.NewPage(req.Page, req.PageSize, len(list)),
	}, nil
}

type nodeMetric struct {
	cpuCapacity          int64
	cpuAllocated         float64
	cpuUsage             float64
	memoryCapacity       int64
	memoryAllocated      float64
	memoryUsage          float64
	gpuCount             int32
	gpuCoreTotal         float64
	gpuCoreUsage         float64
	gpuCoreAllocated     float64
	gpuMemoryTotal       int64
	gpuMemoryAllocatable int64
	gpuMemoryUsage       int64
	gpuMemoryAllocated   int64
}

// ListNodes lists nodes under the specific cluster.
func (h *CoreHandler) ListNodes(ctx context.Context, req *corev1alpha1.ListNodesRequest) (*corev1alpha1.ListNodesResponse, error) {
	if errors := validation.IsValidLabelValue(req.Name); len(errors) > 0 {
		return &corev1alpha1.ListNodesResponse{Pagination: NewEmptyPage(req.Page, req.PageSize)}, nil
	}

	nodes, err := h.service.ListNodes(ctx, req.GetCluster())
	if err != nil {
		return nil, err
	}

	// filter all nodes first.
	filtered := filterNodes(nodes, req.GetName(), req.GetRole(), req.GetPhase())

	semaphore := make(chan struct{}, 10)
	resultChan := make(chan *corev1alpha1.Node, len(filtered))
	for _, node := range filtered {
		go func(node *corev1.Node) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			metric := h.batchGetNodeMetrics(ctx, req.GetCluster(), node)
			nodeProto := convertNode2Proto(node, &metric)
			resultChan <- nodeProto
		}(node)
	}

	items := make([]*corev1alpha1.Node, 0, len(filtered))
	for range filtered {
		items = append(items, <-resultChan)
	}

	if req.GetSortOption().GetField() == "" {
		req.SortOption = &types.SortOption{Field: "metadata.name", Asc: true}
	}
	if err = utils.SortStructSlice(items, req.GetSortOption().GetField(), req.GetSortOption().GetAsc(), utils.SnakeToCamelMapper()); err != nil {
		return nil, err
	}

	paged := utils.PagedItems(items, req.GetPage(), req.GetPageSize())

	return &corev1alpha1.ListNodesResponse{
		Items:      paged,
		Pagination: utils.NewPage(req.GetPage(), req.GetPageSize(), len(filtered)),
	}, nil
}

func (h *CoreHandler) batchGetNodeMetrics(ctx context.Context, cluster string, node *corev1.Node) nodeMetric {
	metrics := nodeMetric{}
	vec, err := h.monitoringService.QueryGPUVector(ctx, cluster, node.Name, "", "", monitoring.GPUQueryTypeCount)
	if err == nil && len(vec) > 0 {
		metrics.gpuCount = int32(vec[0].Value)
	}
	// gpu core metrics
	vec, err = h.monitoringService.QueryNode(ctx, cluster, node.Name, monitoring.QueryTypeGPUCoreTotal)
	if err == nil {
		metrics.gpuCoreTotal += float64(vec[0].Value)
	}
	vec, err = h.monitoringService.QueryNode(ctx, cluster, node.Name, monitoring.QueryTypeGPUCoreUsed)
	if err == nil {
		metrics.gpuCoreUsage += float64(vec[0].Value)
	}
	vec, err = h.monitoringService.QueryNode(ctx, cluster, node.Name, monitoring.QueryTypeGPUCoreAllocated)
	if err == nil {
		metrics.gpuCoreAllocated += float64(vec[0].Value)
	}

	// gpu memory metrics
	vec, err = h.monitoringService.QueryNode(ctx, cluster, node.Name, monitoring.QueryTypeGPUMemoryTotal)
	if err == nil {
		metrics.gpuMemoryAllocatable += int64(vec[0].Value)
		metrics.gpuMemoryTotal += int64(float64(vec[0].Value) / annotations.GetFactorFromAnnotation(node.Annotations))
	}
	vec, err = h.monitoringService.QueryNode(ctx, cluster, node.Name, monitoring.QueryTypeGPUMemoryUsed)
	if err == nil {
		metrics.gpuMemoryUsage = int64(vec[0].Value)
	}
	vec, err = h.monitoringService.QueryNode(ctx, cluster, node.Name, monitoring.QueryTypeGPUMemoryAllocated)
	if err == nil {
		metrics.gpuMemoryAllocated += int64(vec[0].Value)
	}

	return metrics
}

// GetNode gets the details of the specified node.
func (h *CoreHandler) GetNode(ctx context.Context, req *corev1alpha1.GetNodeRequest) (*corev1alpha1.Node, error) {
	if errs := validation.IsDNS1035Label(req.GetCluster()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cluster name %s is invalid, error: %s", req.GetCluster(), errs)
	}

	node, err := h.service.GetNode(ctx, req.GetCluster(), req.GetName())
	if err != nil {
		return nil, err
	}

	group := &errgroup.Group{}

	metrics := &nodeMetric{}

	// cpu core metrics.
	group.Go(func() error {
		vec, err := h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeCPUUsed)
		if err == nil {
			metrics.cpuUsage = float64(vec[0].Value)
		}
		vec, err = h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeCPUAllocated)
		if err == nil {
			metrics.cpuAllocated = float64(vec[0].Value)
		}
		vec, err = h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeCPUTotal)
		if err == nil {
			metrics.cpuCapacity = int64(vec[0].Value)
		}
		return nil
	})

	// memory metrics.
	group.Go(func() error {
		vec, err := h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeMemoryUsed)
		if err == nil {
			metrics.memoryUsage = float64(vec[0].Value)
		}
		vec, err = h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeMemoryAllocated)
		if err == nil {
			metrics.memoryAllocated = float64(vec[0].Value)
		}
		vec, err = h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeMemoryTotal)
		if err == nil {
			metrics.memoryCapacity = int64(vec[0].Value)
		}
		return nil
	})

	// gpu core metrics.
	group.Go(func() error {
		vec, err := h.monitoringService.QueryGPUVector(ctx, req.GetCluster(), node.Name, "", "", monitoring.GPUQueryTypeCount)
		if err == nil && len(vec) > 0 {
			metrics.gpuCount = int32(vec[0].Value)
		}
		vec, err = h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeGPUCoreTotal)
		if err == nil {
			metrics.gpuCoreTotal += float64(vec[0].Value)
		}
		vec, err = h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeGPUCoreUsed)
		if err == nil && len(vec) > 0 {
			metrics.gpuCoreUsage += float64(vec[0].Value)
		}
		vec, err = h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeGPUCoreAllocated)
		if err == nil && len(vec) > 0 {
			metrics.gpuCoreAllocated += float64(vec[0].Value)
		}
		return nil
	})

	// gpu memory metrics.
	vec, err := h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeGPUMemoryTotal)
	if err == nil && len(vec) > 0 {
		metrics.gpuMemoryAllocatable += int64(vec[0].Value)
		metrics.gpuMemoryTotal += int64(float64(vec[0].Value) / annotations.GetFactorFromAnnotation(node.Annotations))
	}
	vec, err = h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeGPUMemoryUsed)
	if err == nil && len(vec) > 0 {
		metrics.gpuMemoryUsage = int64(vec[0].Value)
	}
	vec, err = h.monitoringService.QueryNode(ctx, req.GetCluster(), node.Name, monitoring.QueryTypeGPUMemoryAllocated)
	if err == nil && len(vec) > 0 {
		metrics.gpuMemoryAllocated += int64(vec[0].Value)
	}

	group.Wait()
	return convertNode2Proto(node, metrics), nil
}

func filterNodes(nodes []*corev1.Node, keyword string, role corev1alpha1.Role, phase corev1alpha1.NodePhase) []*corev1.Node {
	res := make([]*corev1.Node, 0, len(nodes))
	for _, node := range nodes {
		if !utils.MatchByFuzzyName(node, keyword) {
			continue
		}

		nodeRole := getNodeRole(node)
		nodePhase := GetNodePhase(node.Status.Conditions)

		if (role == corev1alpha1.Role_NODE_ROLE_UNSPECIFIED || nodeRole == role) &&
			(phase == corev1alpha1.NodePhase_NODE_PHASE_UNSPECIFIED || nodePhase == phase) {
			res = append(res, node)
		}
	}
	return res
}

func getNodeRole(node *corev1.Node) corev1alpha1.Role {
	if _, ok := node.Labels["role.kubernetes.io/control-plane"]; ok {
		return corev1alpha1.Role_CONTROL_PLANE
	}
	return corev1alpha1.Role_WORKER
}

func (h *CoreHandler) PutNodeLabels(ctx context.Context, req *corev1alpha1.PutNodeLabelsRequest) (*corev1alpha1.PutNodeLabelsResponse, error) {
	clusterName := req.GetCluster()
	name := req.GetNode()
	labels := req.GetLabels()

	nodeLabels, err := h.service.PutNodeLabels(ctx, clusterName, name, labels)
	if err != nil {
		return nil, err
	}

	return &corev1alpha1.PutNodeLabelsResponse{Labels: nodeLabels}, err
}

func (h *CoreHandler) PutNodeTaints(ctx context.Context, req *corev1alpha1.PutNodeTaintsRequest) (*corev1alpha1.PutNodeTaintsResponse, error) {
	clusterName := req.GetCluster()
	name := req.GetNode()
	taints := req.GetTaints()

	convertedTaints := []*corev1.Taint{}
	for i := range taints {
		taint := corev1.Taint{
			Key:    taints[i].Key,
			Value:  taints[i].Value,
			Effect: taintEffectConvert(taints[i].Effect),
		}
		convertedTaints = append(convertedTaints, &taint)
	}

	nodeTaints, err := h.service.PutNodeTaints(ctx, clusterName, name, convertedTaints)
	if err != nil {
		return nil, err
	}

	res := &corev1alpha1.PutNodeTaintsResponse{}
	for i := range nodeTaints {
		taint := corev1alpha1.Taint{
			Key:    nodeTaints[i].Key,
			Value:  nodeTaints[i].Value,
			Effect: convertTaintEffect(nodeTaints[i].Effect),
		}
		res.Taints = append(res.Taints, &taint)
	}
	return res, err
}

func (h *CoreHandler) UpdateNodeAnnotations(ctx context.Context, req *corev1alpha1.UpdateNodeAnnotationsRequest) (*corev1alpha1.UpdateNodeAnnotationsResponse, error) {
	clusterName := req.GetCluster()
	name := req.GetNode()
	annotations := req.GetAnnotations()

	nodeAnnotations, err := h.service.UpdateNodeAnnotations(ctx, clusterName, name, annotations)
	if err != nil {
		return nil, err
	}
	return &corev1alpha1.UpdateNodeAnnotationsResponse{Annotations: nodeAnnotations}, err
}

func (h *CoreHandler) UnScheduleNode(ctx context.Context, req *corev1alpha1.ScheduleNodeRequest) (*corev1alpha1.Node, error) {
	node, err := h.service.UnScheduleNode(ctx, req.Cluster, req.Node, true)
	if err != nil {
		return nil, err
	}
	return convertNode2Proto(node, nil), nil
}

func (h *CoreHandler) ScheduleNode(ctx context.Context, req *corev1alpha1.ScheduleNodeRequest) (*corev1alpha1.Node, error) {
	node, err := h.service.UnScheduleNode(ctx, req.Cluster, req.Node, false)
	if err != nil {
		return nil, err
	}
	return convertNode2Proto(node, nil), nil
}

// GetConfigMap gets a configMap under the namespaces of a specific cluster.
func (h *CoreHandler) GetConfigMap(ctx context.Context, req *corev1alpha1.GetConfigMapRequest) (*corev1alpha1.ConfigMap, error) {
	if errs := validation.IsDNS1035Label(req.GetCluster()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cluster name %s is invalid, error: %s", req.GetCluster(), errs)
	}
	if errs := validation.IsDNS1035Label(req.GetNamespace()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "namespace name %s is invalid, error: %s", req.GetNamespace(), errs)
	}
	if errs := validation.IsDNS1035Label(req.GetName()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "configmap name %s is invalid, error: %s", req.GetName(), errs)
	}

	configMap, err := h.service.GetConfigMap(ctx, req.GetCluster(), req.GetNamespace(), req.GetName())
	if err != nil {
		return nil, err
	}
	return convertConfigMap2Proto(*configMap), nil
}

// GetConfigMapJSON gets a configMap json under the namespaces of a specific cluster.
func (h *CoreHandler) GetConfigMapJSON(ctx context.Context, req *corev1alpha1.GetConfigMapJSONRequest) (*corev1alpha1.GetConfigMapJSONResponse, error) {
	if errs := validation.IsDNS1035Label(req.GetCluster()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cluster name %s is invalid, error: %s", req.GetCluster(), errs)
	}
	if errs := validation.IsDNS1035Label(req.GetNamespace()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "namespace name %s is invalid, error: %s", req.GetNamespace(), errs)
	}
	if errs := validation.IsDNS1035Label(req.GetName()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "configmap name %s is invalid, error: %s", req.GetName(), errs)
	}

	configMap, err := h.service.GetConfigMap(ctx, req.GetCluster(), req.GetNamespace(), req.GetName())
	if err != nil {
		return nil, err
	}
	configMap.APIVersion = "v1"
	configMap.Kind = "ConfigMap"

	return &corev1alpha1.GetConfigMapJSONResponse{Data: utils.MarshalObj(configMap)}, nil
}

// UpdateConfigMap updates a configMap under the namespaces of a specific cluster.
func (h *CoreHandler) UpdateConfigMap(ctx context.Context, req *corev1alpha1.UpdateConfigMapRequest) (*corev1alpha1.UpdateConfigMapResponse, error) {
	if errs := validation.IsDNS1035Label(req.GetCluster()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cluster name %s is invalid, error: %s", req.GetCluster(), errs)
	}
	if errs := validation.IsDNS1035Label(req.GetNamespace()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "namespace name %s is invalid, error: %s", req.GetNamespace(), errs)
	}
	if errs := validation.IsDNS1035Label(req.GetName()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "configMap name %s is invalid, error: %s", req.GetName(), errs)
	}

	configmap := &corev1.ConfigMap{}
	if err := json.Unmarshal([]byte(req.Data), configmap); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	msg := "The %s(%s) in the request body is inconsistent with that in the URL."
	if configmap.Namespace != req.Namespace {
		return nil, status.Errorf(codes.InvalidArgument, msg, "namespace", configmap.Namespace)
	}
	if configmap.Name != req.Name {
		return nil, status.Errorf(codes.InvalidArgument, msg, "name", configmap.Name)
	}

	configMap, err := h.service.UpdateConfigMap(ctx, req.GetCluster(), req.GetNamespace(), configmap)
	if err != nil {
		return nil, err
	}

	return &corev1alpha1.UpdateConfigMapResponse{Data: utils.MarshalObj(configMap)}, nil
}
