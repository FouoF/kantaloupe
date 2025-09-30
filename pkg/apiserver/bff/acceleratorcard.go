package bff

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/dlclark/regexp2"
	prometheusmodel "github.com/prometheus/common/model"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/klog/v2"

	acceleratorcardv1alpha1 "github.com/dynamia-ai/kantaloupe/api/acceleratorcard/v1alpha1"
	clustersv1alpha1 "github.com/dynamia-ai/kantaloupe/api/clusters/v1alpha1"
	corev1alpha1 "github.com/dynamia-ai/kantaloupe/api/core/v1alpha1"
	kantaloupeapi "github.com/dynamia-ai/kantaloupe/api/v1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/service/cluster"
	"github.com/dynamia-ai/kantaloupe/pkg/service/core"
	"github.com/dynamia-ai/kantaloupe/pkg/service/monitoring"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/annotations"
)

var _ kantaloupeapi.AcceleratorCardServer = &AcceleratorCardHandler{}

type AcceleratorCardHandler struct {
	sync.Mutex
	kantaloupeapi.UnimplementedAcceleratorCardServer
	clusterService    cluster.Service
	workloadService   core.Service
	monitoringService monitoring.Service
}

type AcceleratorCardMetricsHandler interface {
	ListAcceleratorCard(ctx context.Context, req *acceleratorcardv1alpha1.ListAcceleratorCardsRequest, handler *AcceleratorCardHandler) (*acceleratorcardv1alpha1.ListAcceleratorCardsResponse, error)
	GetAcceleratorCard(ctx context.Context, req *acceleratorcardv1alpha1.GetAcceleratorCardRequest, handler *AcceleratorCardHandler) (*acceleratorcardv1alpha1.AcceleratorCard, error)
	ListModelNames(ctx context.Context, req *acceleratorcardv1alpha1.ListModelNamesRequest, handler *AcceleratorCardHandler) (*acceleratorcardv1alpha1.ListModelNamesResponse, error)
}

func NewAcceleratorCardHandler(clientManager engine.ClientManagerInterface, prometheus engine.PrometheusInterface) *AcceleratorCardHandler {
	return &AcceleratorCardHandler{
		clusterService:    cluster.NewService(clientManager),
		monitoringService: monitoring.NewService(prometheus),
		workloadService:   core.NewService(clientManager),
	}
}

func (h *AcceleratorCardHandler) ListAcceleratorCard(ctx context.Context, req *acceleratorcardv1alpha1.ListAcceleratorCardsRequest) (*acceleratorcardv1alpha1.ListAcceleratorCardsResponse, error) {
	if errs := validation.IsDNS1035Label(req.GetCluster()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cluster name %s is invalid, error: %s", req.GetCluster(), errs)
	}

	res, err := h.listAcceleratorCard(ctx, req)
	if err != nil {
		return nil, err
	}

	// list all nodes, as the number of nodes is small, we can tolerate the performance loss.
	nodes, err := h.workloadService.ListNodes(ctx, req.GetCluster())
	if err != nil {
		klog.ErrorS(err, "failed to list nodes", "Error:", err)
		return nil, err
	}
	nodeMap := map[string]*corev1.Node{}
	for _, node := range nodes {
		nodeMap[node.Name] = node
	}

	// add node addresses to cards.
	for _, card := range res.Items {
		node, ok := nodeMap[card.Node]
		if !ok {
			continue
		}
		card.GpuMemoryTotal = card.GpuMemoryAllocatable / int64(annotations.GetFactorFromAnnotation(node.Annotations))
		address := []*corev1alpha1.NodeAddress{}
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeHostName {
				continue
			}
			address = append(address, &corev1alpha1.NodeAddress{
				Type:    string(addr.Type),
				Address: addr.Address,
			})
		}
		card.NodeAddresses = address
	}
	return res, nil
}

func (h *AcceleratorCardHandler) GetAcceleratorCard(ctx context.Context, req *acceleratorcardv1alpha1.GetAcceleratorCardRequest) (*acceleratorcardv1alpha1.AcceleratorCard, error) {
	if errs := validation.IsDNS1035Label(req.GetCluster()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cluster name %s is invalid, error: %s", req.GetCluster(), errs)
	}

	cluster, err := h.clusterService.GetCluster(ctx, req.GetCluster())
	if err != nil {
		return nil, err
	}

	res, err := h.getAcceleratorCard(ctx, req)
	if err != nil {
		return nil, err
	}

	// set cluster provider and type to card
	if provider, ok := clustersv1alpha1.ClusterProvider_value[cluster.Spec.Provider]; ok {
		res.Provider = clustersv1alpha1.ClusterProvider(provider)
	}
	if typ, ok := clustersv1alpha1.ClusterType_value[cluster.Spec.Type]; ok {
		res.Type = clustersv1alpha1.ClusterType(typ)
	}
	node, err := h.workloadService.GetNode(ctx, req.GetCluster(), req.GetNode())
	if err != nil {
		return nil, err
	}
	limit, err := calculateWorkloadLimits(node, cluster.Spec.Type)
	res.GpuMemoryTotal = res.GpuMemoryAllocatable / int64(annotations.GetFactorFromAnnotation(node.Annotations))
	if err != nil {
		return nil, err
	}
	res.WorkloadLimit = int32(limit)
	return res, nil
}

func (h *AcceleratorCardHandler) ListModelNames(ctx context.Context, req *acceleratorcardv1alpha1.ListModelNamesRequest) (*acceleratorcardv1alpha1.ListModelNamesResponse, error) {
	if errs := validation.IsDNS1035Label(req.GetCluster()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cluster name %s is invalid, error: %s", req.GetCluster(), errs)
	}

	tempVec, err := h.monitoringService.QueryGPUVector(ctx, req.GetCluster(), "", "", "", monitoring.GPUQueryTypeTemperature)
	if err != nil && len(tempVec) == 0 {
		return nil, err
	}
	modelNames := make(map[string]bool, 0)
	for _, vec := range tempVec {
		if modelName, ok := vec.Metric["modelName"]; ok {
			modelNames[string(modelName)] = true
		}
	}
	Items := make([]string, 0)
	for name := range modelNames {
		Items = append(Items, name)
	}
	return &acceleratorcardv1alpha1.ListModelNamesResponse{ModelNames: Items}, nil
}

func filterCards(cards []*acceleratorcardv1alpha1.AcceleratorCard, keyword, state string) []*acceleratorcardv1alpha1.AcceleratorCard {
	ret := []*acceleratorcardv1alpha1.AcceleratorCard{}
	for idx := range cards {
		card := cards[idx]

		// TODO: Card not have GetName function.
		regexp, err := regexp2.Compile(keyword, utils.Regex)
		if err != nil {
			continue
		}
		match, err := regexp.MatchString(card.Uuid)
		if err != nil || !match {
			continue
		}

		if state != acceleratorcardv1alpha1.AcceleratorCardState_ACCELERATORCARD_STATE_UNSPECIFIED.String() &&
			state != card.State.String() {
			continue
		}
		ret = append(ret, card)
	}
	return ret
}

func calculateWorkloadLimits(node *corev1.Node, clusterType string) (int, error) {
	var n int
	var err error
	if clusterType == clustersv1alpha1.ClusterType_METAX.String() {
		return 16, nil
	}
	if t, ok := node.Labels[annotations.NodeNVIDIAGPUAnnotation]; ok {
		values := strings.Split(t, ",")
		if len(values) < 2 {
			klog.InfoS("invalid node GPU annotation", "annotation", t)
			return 0, nil
		}
		n, err = strconv.Atoi(values[1])
		if err != nil {
			return 0, err
		}
	}
	return n, nil
}

func (h *AcceleratorCardHandler) getAcceleratorCard(ctx context.Context, req *acceleratorcardv1alpha1.GetAcceleratorCardRequest) (*acceleratorcardv1alpha1.AcceleratorCard, error) {
	tempVec, err := h.monitoringService.QueryGPU(ctx, req.Cluster, req.Uuid, monitoring.GPUQueryTypeTemperature)
	if err != nil {
		return nil, err
	}

	var (
		memoryUsageVec, memoryAllocatedVec, memoryAllocatableVec          prometheusmodel.Vector
		coreTotoalVec, coreUsageVec, coreAllocatedVec, powerVec, errorVec prometheusmodel.Vector
	)
	group := &errgroup.Group{}
	group.Go(func() error {
		memoryAllocatableVec, err = h.monitoringService.QueryGPU(ctx, req.GetCluster(), req.GetUuid(), monitoring.GPUQueryTypeMemoryTotal)
		if err != nil {
			klog.ErrorS(err, "failed to query memory total", "uuid", req.GetUuid())
		}
		memoryUsageVec, err = h.monitoringService.QueryGPU(ctx, req.GetCluster(), req.GetUuid(), monitoring.GPUQueryTypeMemoryUsed)
		if err != nil {
			klog.ErrorS(err, "failed to query memory used", "uuid", req.GetUuid())
		}
		memoryAllocatedVec, err = h.monitoringService.QueryGPU(ctx, req.GetCluster(), req.GetUuid(), monitoring.GPUQueryTypeMemoryAllocated)
		if err != nil {
			klog.ErrorS(err, "failed to query memory allocated", "uuid", req.GetUuid())
		}
		return nil
	})

	group.Go(func() error {
		powerVec, err = h.monitoringService.QueryGPU(ctx, req.GetCluster(), req.GetUuid(), monitoring.GPUQueryTypePower)
		if err != nil {
			klog.ErrorS(err, "failed to query power", "uuid", req.GetUuid())
		}
		errorVec, err = h.monitoringService.QueryGPU(ctx, req.GetCluster(), req.GetUuid(), monitoring.GPUQueryTypeErrors)
		if err != nil {
			klog.ErrorS(err, "failed to query error", "uuid", req.GetUuid())
		}
		coreTotoalVec, err = h.monitoringService.QueryGPU(ctx, req.GetCluster(), req.GetUuid(), monitoring.GPUQueryTypeCoreTotal)
		if err != nil {
			klog.ErrorS(err, "failed to query core total", "uuid", req.GetUuid())
		}
		coreUsageVec, err = h.monitoringService.QueryGPU(ctx, req.GetCluster(), req.GetUuid(), monitoring.GPUQueryTypeCoreUsed)
		if err != nil {
			klog.ErrorS(err, "failed to query core usage", "uuid", req.GetUuid())
		}
		coreAllocatedVec, err = h.monitoringService.QueryGPU(ctx, req.GetCluster(), req.GetUuid(), monitoring.GPUQueryTypeCoreAllocated)
		if err != nil {
			klog.ErrorS(err, "failed to query core allocated", "uuid", req.GetUuid())
		}
		return nil
	})

	group.Wait()
	card := generateAcceleratorCard(tempVec, powerVec, errorVec,
		memoryAllocatableVec, memoryUsageVec, memoryAllocatedVec,
		coreTotoalVec, coreUsageVec, coreAllocatedVec,
		"UUID",
	)

	return card, nil
}

// ListAcceleratorCard implements AcceleratorCardMetrics.
func (h *AcceleratorCardHandler) listAcceleratorCard(ctx context.Context, req *acceleratorcardv1alpha1.ListAcceleratorCardsRequest) (*acceleratorcardv1alpha1.ListAcceleratorCardsResponse, error) {
	var tempVec prometheusmodel.Vector
	var err error
	if req.GetNode() == constants.SelectAll {
		req.Node = ""
	}
	if req.GetModel() == constants.SelectAll {
		req.Model = ""
	}
	tempVec, err = h.monitoringService.QueryGPUVector(ctx, req.GetCluster(), req.GetNode(), "", req.GetModel(), monitoring.GPUQueryTypeTemperature)
	if err != nil && len(tempVec) == 0 {
		return nil, err
	}

	var (
		memoryUsageVec, memoryAllocatedVec, memoryAllocatableVec          prometheusmodel.Vector
		coreTotoalVec, coreUsageVec, coreAllocatedVec, powerVec, errorVec prometheusmodel.Vector
	)
	group := &errgroup.Group{}
	group.Go(func() error {
		memoryAllocatableVec, err = h.monitoringService.QueryGPUVector(ctx, req.GetCluster(), "", "", "", monitoring.GPUQueryTypeMemoryTotal)
		if err != nil {
			klog.ErrorS(err, "failed to query GPU memory total")
		}
		memoryUsageVec, err = h.monitoringService.QueryGPUVector(ctx, req.GetCluster(), "", "", "", monitoring.GPUQueryTypeMemoryUsed)
		if err != nil {
			klog.ErrorS(err, "failed to query GPU memory used")
		}
		memoryAllocatedVec, err = h.monitoringService.QueryGPUVector(ctx, req.GetCluster(), "", "", "", monitoring.GPUQueryTypeMemoryAllocated)
		if err != nil {
			klog.ErrorS(err, "failed to query GPU memory allocated")
		}
		return nil
	})

	group.Go(func() error {
		errorVec, err = h.monitoringService.QueryGPUVector(ctx, req.GetCluster(), "", "", "", monitoring.GPUQueryTypeErrors)
		if err != nil {
			klog.ErrorS(err, "failed to query GPU errors")
		}
		powerVec, err = h.monitoringService.QueryGPUVector(ctx, req.GetCluster(), "", "", "", monitoring.GPUQueryTypePower)
		if err != nil {
			klog.ErrorS(err, "failed to query GPU power")
		}
		coreTotoalVec, err = h.monitoringService.QueryGPUVector(ctx, req.GetCluster(), "", "", "", monitoring.GPUQueryTypeCoreTotal)
		if err != nil {
			klog.ErrorS(err, "failed to query GPU core total")
		}
		coreUsageVec, err = h.monitoringService.QueryGPUVector(ctx, req.GetCluster(), "", "", "", monitoring.GPUQueryTypeCoreUsed)
		if err != nil {
			klog.ErrorS(err, "failed to query GPU core usage")
		}
		coreAllocatedVec, err = h.monitoringService.QueryGPUVector(ctx, req.GetCluster(), "", "", "", monitoring.GPUQueryTypeCoreAllocated)
		if err != nil {
			klog.ErrorS(err, "failed to query GPU core allocated")
		}
		return nil
	})
	group.Wait()

	cards := []*acceleratorcardv1alpha1.AcceleratorCard{}
	for _, vec := range tempVec {
		if uuid, ok := vec.Metric["UUID"]; ok {
			res := generateAcceleratorCard(
				getVectorForUUID(string(uuid), tempVec),
				getVectorForUUID(string(uuid), powerVec),
				getVectorForUUID(string(uuid), errorVec),
				getVectorForUUID(string(uuid), memoryAllocatableVec),
				getVectorForUUID(string(uuid), memoryUsageVec),
				getVectorForUUID(string(uuid), memoryAllocatedVec),
				getVectorForUUID(string(uuid), coreTotoalVec),
				getVectorForUUID(string(uuid), coreUsageVec),
				getVectorForUUID(string(uuid), coreAllocatedVec),
				"UUID",
			)
			cards = append(cards, res)
		}
	}

	filtered := filterCards(cards, req.GetUuid(), req.GetState().String())
	if err := utils.SortStructSlice(filtered, req.GetSortOption().GetField(), req.GetSortOption().GetAsc(), utils.SnakeToCamelMapper()); err != nil {
		return nil, err
	}

	paged := utils.PagedItems(filtered, req.Page, req.PageSize)
	return &acceleratorcardv1alpha1.ListAcceleratorCardsResponse{
		Items:      paged,
		Pagination: utils.NewPage(req.Page, req.PageSize, len(filtered)),
	}, nil
}

func getVectorForUUID(uuid string, vec prometheusmodel.Vector) prometheusmodel.Vector {
	res := prometheusmodel.Vector{}
	// nvidia, mx, ascend.
	idKeys := []prometheusmodel.LabelName{"UUID", "deviceuuid", "uuid", "vdie_id"}
	for _, v := range vec {
		for _, key := range idKeys {
			if val, ok := v.Metric[key]; ok && uuid == string(val) {
				res = append(res, v)
				break // Found a match for this sample, move to the next one
			}
		}
	}
	return res
}

func generateAcceleratorCard(
	tempVec, powerVec, errorVec, memoryAllocatableVec,
	memoryUsageVec, memoryAllocatedVec, coreTotoalVec,
	coreUsageVec, coreAllocatedVec prometheusmodel.Vector,
	id string,
) *acceleratorcardv1alpha1.AcceleratorCard {
	res := &acceleratorcardv1alpha1.AcceleratorCard{}

	if len(tempVec) > 0 {
		res.Uuid = string(tempVec[0].Metric[prometheusmodel.LabelName(id)])
		res.Model = string(tempVec[0].Metric["modelName"])
		res.Node = string(tempVec[0].Metric["node"])
		res.Temperature = float64(tempVec[0].Value)
		if res.Temperature == -1 {
			res.Temperature = 0
		}
		// TODO:
		res.State = acceleratorcardv1alpha1.AcceleratorCardState_HEALTH
	}
	if len(powerVec) > 0 {
		res.Power = float64(powerVec[0].Value)
	}
	if len(errorVec) > 0 && int(errorVec[0].Value) > 0 {
		res.State = acceleratorcardv1alpha1.AcceleratorCardState_ERROR
	}
	if len(memoryAllocatableVec) > 0 {
		res.GpuMemoryAllocatable = int64(memoryAllocatableVec[0].Value)
	}
	if len(memoryUsageVec) > 0 {
		res.GpuMemoryUsage = int64(memoryUsageVec[0].Value)
	}
	if len(memoryAllocatedVec) > 0 {
		res.GpuMemoryAllocated = int64(memoryAllocatedVec[0].Value)
	}
	if len(coreTotoalVec) > 0 {
		res.GpuCoreTotal = int32(coreTotoalVec[0].Value)
	}
	if len(coreUsageVec) > 0 {
		res.GpuCoreUsage = int32(coreUsageVec[0].Value)
	}
	if len(coreAllocatedVec) > 0 {
		res.GpuCoreAllocated = int32(coreAllocatedVec[0].Value)
	}

	return res
}
