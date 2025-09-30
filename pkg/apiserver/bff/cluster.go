package bff

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"sort"
	"strings"
	"sync"

	prometheusmodel "github.com/prometheus/common/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/klog/v2"

	clustersv1alpha1 "github.com/dynamia-ai/kantaloupe/api/clusters/v1alpha1"
	clustercrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
	monitoringv1alpha1 "github.com/dynamia-ai/kantaloupe/api/monitoring/v1alpha1"
	kantaloupeapi "github.com/dynamia-ai/kantaloupe/api/v1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/service/cluster"
	"github.com/dynamia-ai/kantaloupe/pkg/service/core"
	"github.com/dynamia-ai/kantaloupe/pkg/service/monitoring"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/helper"
)

var (
	_ kantaloupeapi.ClusterServer = &ClusterHandler{}

	resourceTypes = []monitoringv1alpha1.ResourceType{
		monitoringv1alpha1.ResourceType_RESOURCE_TYPE_CPU,
		monitoringv1alpha1.ResourceType_RESOURCE_TYPE_MEMORY,
		monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_CORE,
		monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_MEMORY,
	}
)

type ClusterHandler struct {
	sync.Mutex
	kantaloupeapi.UnimplementedClusterServer
	clusterService    cluster.Service
	coreService       core.Service
	monitoringService monitoring.Service
}

// NewClusterHandler new a cluster handler by client manager.
func NewClusterHandler(clientManager engine.ClientManagerInterface, prometheus engine.PrometheusInterface) *ClusterHandler {
	return &ClusterHandler{
		clusterService:    cluster.NewService(clientManager),
		coreService:       core.NewService(clientManager),
		monitoringService: monitoring.NewService(prometheus),
	}
}

// ListClusters lists clusters cr resource and convert to protobuf clusters.
func (h *ClusterHandler) ListClusters(
	ctx context.Context,
	req *clustersv1alpha1.ListClustersRequest,
) (*clustersv1alpha1.ListClustersResponse, error) {
	// 1. list all clusters.
	clusters, err := h.clusterService.ListClusters(ctx)
	if err != nil {
		klog.ErrorS(err, "failed to list cluster", "Error:", err)
		return nil, err
	}

	// 1. we filter all clusters first.
	// 2. sort all clusters.
	// 3. page all clusters.
	filtered := filterClusters(clusters, req.GetName(), req.GetType().String(), req.GetState().String(), req.GetProvider().String())
	// make local-cluster the first one
	for i, cluster := range filtered {
		if cluster.Name == "local-cluster" {
			// replace it with lowest string
			filtered[i].Name = "!"
			break
		}
	}
	if err = utils.SortStructSlice(filtered, req.GetSortOption().GetField(), req.GetSortOption().GetAsc(), utils.SnakeToCamelMapper()); err != nil {
		return nil, err
	}
	for i, cluster := range filtered {
		if cluster.Name == "!" {
			// convert the name back
			filtered[i].Name = "local-cluster"
			break
		}
	}
	paged := utils.PagedItems(filtered, req.Page, req.PageSize)

	// 4. enrich clusters with metrics and convert cluster.
	items := h.enrichClustersWithMetrics(ctx, paged)

	return &clustersv1alpha1.ListClustersResponse{
		Items:      items,
		Pagination: utils.NewPage(req.Page, req.PageSize, len(filtered)),
	}, nil
}

func filterClusters(clusters []clustercrdv1alpha1.Cluster, fuzzyName, clusterType, clusterState, provider string) []*clustercrdv1alpha1.Cluster {
	res := []*clustercrdv1alpha1.Cluster{}

	for idx := range clusters {
		cluster := clusters[idx]

		if !utils.MatchByFuzzyName(&cluster, fuzzyName) {
			continue
		}

		if clusterState != "" && clusterState != clustersv1alpha1.ClusterState_UNSPECIFED.String() {
			state := convertCondition2State(cluster.Status)
			if clusterState != "" && clusterState != state.String() {
				continue
			}
		}
		if clusterType != "" && clusterType != clustersv1alpha1.ClusterType_CLUSTER_TYPE_UNSPECIFIED.String() {
			if clusterType != cluster.Spec.Type {
				continue
			}
		}
		if provider != "" && provider != clustersv1alpha1.ClusterProvider_CLUSTER_PROVIDER_UNSPECIFIED.String() {
			if provider != cluster.Spec.Provider {
				continue
			}
		}
		res = append(res, &cluster)
	}
	return res
}

func (h *ClusterHandler) enrichClustersWithMetrics(ctx context.Context, clusters []*clustercrdv1alpha1.Cluster) []*clustersv1alpha1.Cluster {
	type clusterResult struct {
		index   int
		cluster *clustersv1alpha1.Cluster
		err     error
	}

	resultChan := make(chan clusterResult, len(clusters))
	semaphore := make(chan struct{}, 10)

	for i, cluster := range clusters {
		go func(idx int, c *clustercrdv1alpha1.Cluster) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := clusterResult{index: idx}

			if !helper.IsClusterReady(&c.Status) {
				result.cluster = ConvertCluster2Proto(c, nil)
				resultChan <- result
				return
			}

			metric, gpuTotal := h.getClusterMetricsAndGPU(ctx, c)

			protoCluster := ConvertCluster2Proto(c, &metric)
			protoCluster.Status.GpuTotal = gpuTotal

			result.cluster = protoCluster
			resultChan <- result
		}(i, cluster)
	}

	items := make([]*clustersv1alpha1.Cluster, len(clusters))
	for range clusters {
		result := <-resultChan
		if result.err != nil {
			klog.ErrorS(result.err, "failed to get cluster metrics", "cluster", clusters[result.index].Name)
		}
		items[result.index] = result.cluster
	}

	return items
}

func (h *ClusterHandler) getClusterMetricsAndGPU(ctx context.Context, cluster *clustercrdv1alpha1.Cluster) (clusterMetric, int32) {
	type metricResult struct {
		resourceType monitoringv1alpha1.ResourceType
		usedVec      prometheusmodel.Vector
		allocatedVec prometheusmodel.Vector
	}

	var gpuTotal int32
	metric := clusterMetric{}
	metricChan := make(chan metricResult, 4)

	for _, rt := range resourceTypes {
		go func(resourceType monitoringv1alpha1.ResourceType) {
			queries := ConvertResourceType2QueryType(resourceType)

			usedChan := make(chan prometheusmodel.Vector, 1)
			allocatedChan := make(chan prometheusmodel.Vector, 1)
			total, err := h.monitoringService.QueryCluster(ctx, cluster.Name, queries["total"])
			if err != nil {
				klog.ErrorS(err, "Error query cluster total", "resourceType", resourceType)
				metricChan <- metricResult{resourceType, prometheusmodel.Vector{}, prometheusmodel.Vector{}}
				return
			}
			if total[0].Value == 0 {
				metricChan <- metricResult{resourceType, prometheusmodel.Vector{}, prometheusmodel.Vector{}}
				return
			}

			go func() {
				vec, err := h.monitoringService.QueryCluster(ctx, cluster.Name, queries["used"])
				if err != nil {
					klog.V(4).ErrorS(err, "failed to query used metrics", "cluster", cluster.GetName(), "type", resourceType.String())
					usedChan <- prometheusmodel.Vector{}
					return
				}
				usedChan <- prometheusmodel.Vector{&prometheusmodel.Sample{Metric: total[0].Metric, Value: vec[0].Value / total[0].Value * 100}}
			}()

			// skip allocated query for cpu and memory.
			if resourceType == monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_CORE ||
				resourceType == monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_MEMORY {
				go func() {
					vec, err := h.monitoringService.QueryCluster(ctx, cluster.Name, queries["allocated"])
					if err != nil {
						klog.V(4).ErrorS(err, "failed to query allocated metrics", "cluster", cluster.GetName(), "type", resourceType.String())
						allocatedChan <- prometheusmodel.Vector{}
						return
					}
					allocatedChan <- prometheusmodel.Vector{&prometheusmodel.Sample{Metric: vec[0].Metric, Value: vec[0].Value / total[0].Value * 100}}
				}()
			} else {
				allocatedChan <- prometheusmodel.Vector{}
			}

			usedVec := <-usedChan
			allocatedVec := <-allocatedChan
			metricChan <- metricResult{resourceType, usedVec, allocatedVec}
		}(rt)
	}

	gpuChan := make(chan int32, 1)
	go func() {
		gpuNum, err := h.monitoringService.QueryGPUVector(ctx, cluster.Name, "", "", "", monitoring.GPUQueryTypeCount)
		if err != nil {
			klog.V(4).ErrorS(err, "failed to query GPU count", "cluster", cluster.GetName())
			gpuChan <- 0
			return
		}

		if len(gpuNum) > 0 {
			gpuChan <- int32(gpuNum[0].Value)
		} else {
			gpuChan <- 0
		}
	}()

	for range resourceTypes {
		result := <-metricChan
		metric.Save(result.resourceType, result.usedVec, result.allocatedVec)
	}

	gpuTotal = <-gpuChan
	return metric, gpuTotal
}

// IntegrateCluster integrates a cluster.
func (h *ClusterHandler) IntegrateCluster(
	ctx context.Context,
	req *clustersv1alpha1.IntegrateClusterRequest,
) (*clustersv1alpha1.Cluster, error) {
	// validate the important params.
	if req.GetKubeConfig() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "the kubeconfig can not be empty")
	}
	if errs := validation.IsDNS1035Label(req.GetName()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cluster name %s is invalid, error: %s", req.GetName(), errs)
	}
	if !utils.IsValidLabelNames(req.GetLabels()) {
		return nil, status.Errorf(codes.InvalidArgument, "cluster label %s is invalid", req.GetLabels())
	}
	if !utils.IsValidAnnotationNames(req.GetAnnotations()) {
		return nil, status.Errorf(codes.InvalidArgument, "cluster annotation %s is invalid", req.GetAnnotations())
	}
	if req.GetType() == clustersv1alpha1.ClusterType_CLUSTER_TYPE_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "cluster type must be specified")
	}

	// validate whether the kubeconfig is valid.
	valid, err := h.clusterService.ValidateKubeconfig(ctx, req.GetKubeConfig())
	if err != nil || !valid {
		return nil, status.Errorf(codes.InvalidArgument, "the kubeconfig is invalid, err: %v", err)
	}

	// validate whether the prometheusAddress is valid if provided.
	if req.GetPrometheusAddress() != "" {
		valid, err := h.clusterService.ValidatePrometheusAddress(ctx, req.GetPrometheusAddress())
		if err != nil || !valid {
			return nil, status.Errorf(codes.InvalidArgument, "the prometheus address is invalid, err: %v", err)
		}
	}

	// validate whether the gatewayAddress is valid if provided.
	if req.GetGatewayAddress() != "" {
		_, err := url.Parse(req.GetGatewayAddress())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "the gateway address is invalid, err: %v", err)
		}
	}

	created, err := h.clusterService.CreateCluster(ctx, buildClusterFromRequest(req), req.GetKubeConfig())
	if err != nil {
		return nil, err
	}

	return ConvertCluster2Proto(created, nil), nil
}

// GetCluster gets the details of the specified cluster.
func (h *ClusterHandler) GetCluster(
	ctx context.Context,
	req *clustersv1alpha1.GetClusterRequest,
) (*clustersv1alpha1.Cluster, error) {
	if errs := validation.IsDNS1035Label(req.GetName()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cluster name %s is invalid, error: %s", req.GetName(), errs)
	}

	cluster, err := h.clusterService.GetCluster(ctx, req.GetName())
	if err != nil {
		klog.ErrorS(err, "failed to get cluster")
		return nil, err
	}
	var gpuNum prometheusmodel.Vector
	var gpuMem prometheusmodel.Vector

	gpuNum, _ = h.monitoringService.QueryGPUVector(context.TODO(), req.GetName(), "", "", "", monitoring.GPUQueryTypeCount)
	gpuMem, _ = h.monitoringService.QueryCluster(context.TODO(), req.GetName(), monitoring.QueryTypeGPUMemoryTotal)
	ret := ConvertCluster2Proto(cluster, nil)
	if gpuNum.Len() > 0 {
		ret.Status.GpuTotal = int32(gpuNum[0].Value)
	}
	if gpuMem.Len() > 0 {
		ret.Status.GpuMemoryTotal = int64(gpuMem[0].Value)
	}
	return ret, nil
}

func (h *ClusterHandler) UpdateCluster(ctx context.Context, req *clustersv1alpha1.UpdateClusterRequest) (*emptypb.Empty, error) {
	if !utils.IsValidLabelNames(req.GetLabels()) {
		return nil, status.Errorf(codes.InvalidArgument, "cluster label %s is invalid", req.GetLabels())
	}
	if !utils.IsValidAnnotationNames(req.GetAnnotations()) {
		return nil, status.Errorf(codes.InvalidArgument, "cluster annotation %s is invalid", req.GetAnnotations())
	}
	cluster, err := h.clusterService.GetCluster(ctx, req.GetName())
	if err != nil {
		return nil, err
	}
	if req.AliasName != "" {
		if cluster.Annotations == nil {
			cluster.Annotations = make(map[string]string)
		}
		cluster.Annotations[constants.ClusterAliasAnnotationKey] = req.AliasName
	}
	if req.GetLabels() != nil {
		if cluster.Labels == nil {
			cluster.Labels = make(map[string]string)
		}
		maps.Copy(cluster.Labels, req.GetLabels())
	}
	if req.GetAnnotations() != nil {
		if cluster.Annotations == nil {
			cluster.Annotations = make(map[string]string)
		}
		maps.Copy(cluster.Annotations, req.GetAnnotations())
	}
	if req.GetDescription() != "" {
		if cluster.Annotations == nil {
			cluster.Annotations = make(map[string]string)
		}
		cluster.Annotations[constants.ClusterDescriptionAnnotationKey] = req.Description
	}
	if req.GetPrometheusAddress() != "" {
		cluster.Spec.PrometheusAddress = req.GetPrometheusAddress()
	}
	if req.GetGatewayAddress() != "" {
		cluster.Spec.GatewayAddress = req.GetGatewayAddress()
	}
	if _, err := h.clusterService.UpdateCluster(ctx, cluster, req.GetKubeConfig()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// DeleteCluster deletes the specified cluster.
func (h *ClusterHandler) DeleteCluster(ctx context.Context, req *clustersv1alpha1.DeleteClusterRequest) (*emptypb.Empty, error) {
	if errs := validation.IsDNS1035Label(req.GetName()); len(errs) != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cluster name %s is invalid, error: %s", req.GetName(), errs)
	}

	if err := h.clusterService.DeleteCluster(ctx, req.GetName()); err != nil {
		klog.ErrorS(err, "failed to delete cluster")
		return &emptypb.Empty{}, err
	}

	return &emptypb.Empty{}, nil
}

// ValidateKubeconfig validats whether the config is valid.
func (h *ClusterHandler) ValidateKubeconfig(
	ctx context.Context,
	req *clustersv1alpha1.ValidateKubeconfigRequest,
) (*clustersv1alpha1.ValidateKubeconfigResponse, error) {
	if req.GetKubeconfig() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "the kubeconfig can not be empty")
	}

	valid, err := h.clusterService.ValidateKubeconfig(ctx, req.GetKubeconfig())
	if err != nil {
		return nil, err
	}

	return &clustersv1alpha1.ValidateKubeconfigResponse{Validate: valid}, nil
}

func (h *ClusterHandler) GetPlatformSummury(
	ctx context.Context,
	req *clustersv1alpha1.GetPlatformSummuryRequest,
) (*clustersv1alpha1.PlatformSummury, error) {
	clusters, err := h.clusterService.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	resp := &clustersv1alpha1.PlatformSummury{
		AcceleratorCardSummury: []*clustersv1alpha1.AcceleratorCardSummury{},
	}

	temp := map[string]*clustersv1alpha1.AcceleratorCardSummury{}
	for _, cluster := range clusters {
		resp.ClusterNum += 1
		resp.NodeNum += cluster.Status.NodeSummary.TotalNum
		resp.KantaloupeflowNum += cluster.Status.KantaloupeflowSummary.TotalNum
		if convertCondition2State(cluster.Status) != clustersv1alpha1.ClusterState_RUNNING {
			continue
		}

		var coreUsedVec prometheusmodel.Vector
		// TODO: use goroutine
		coreUsedVec, err = h.monitoringService.QueryGPUVector(ctx, cluster.Name, "", "", "", monitoring.GPUQueryTypeCoreUsed)
		if err != nil {
			return nil, err
		}
		resp.AcceleratorCardNum += int32(len(coreUsedVec))
		memoryUsedMap, err := h.monitoringService.QueryGPUMap(ctx, cluster.Name, "", "", "", "UUID", monitoring.GPUQueryTypeMemoryUsed)
		if err != nil {
			return nil, err
		}
		memoryAllocatedMap, err := h.monitoringService.QueryGPUMap(ctx, cluster.Name, "", "", "", "UUID", monitoring.GPUQueryTypeMemoryAllocated)
		if err != nil {
			return nil, err
		}
		coreAllocatedMap, err := h.monitoringService.QueryGPUMap(ctx, cluster.Name, "", "", "", "UUID", monitoring.GPUQueryTypeCoreAllocated)
		if err != nil {
			return nil, err
		}
		memoryTotalMap, err := h.monitoringService.QueryGPUMap(ctx, cluster.Name, "", "", "", "UUID", monitoring.GPUQueryTypeMemoryTotal)
		if err != nil {
			return nil, err
		}
		// set threshold to default value.
		if req.GetThreshold() <= 0 {
			req.Threshold = 5
		}
		for _, vec := range coreUsedVec {
			uuid := string(vec.Metric[prometheusmodel.LabelName("UUID")])
			mode := string(vec.Metric["modelName"])
			if _, ok := temp[mode]; !ok {
				temp[mode] = &clustersv1alpha1.AcceleratorCardSummury{Mode: mode}
			}
			var memoryUsage prometheusmodel.SampleValue
			_, ok1 := memoryUsedMap[uuid]
			_, ok2 := memoryTotalMap[uuid]
			if ok1 && ok2 {
				memoryUsage = memoryUsedMap[uuid].Value / memoryTotalMap[uuid].Value * 100
			}

			var idel, used int32
			_, ok1 = memoryAllocatedMap[uuid]
			_, ok2 = coreAllocatedMap[uuid]
			if ok1 && ok2 && (memoryAllocatedMap[uuid].Value > 0 || coreAllocatedMap[uuid].Value > 0) {
				used++
				if vec.Value < prometheusmodel.SampleValue(req.GetThreshold()) && memoryUsage < prometheusmodel.SampleValue(req.GetThreshold()) {
					idel++
				}
			}
			temp[mode].IdelNum += idel
			temp[mode].UseageNum += used
			temp[mode].TotalNum++
		}
	}

	for _, val := range temp {
		resp.AcceleratorCardSummury = append(resp.AcceleratorCardSummury, val)
	}
	sort.Slice(resp.AcceleratorCardSummury, func(i, j int) bool {
		if resp.AcceleratorCardSummury[i].IdelNum == resp.AcceleratorCardSummury[j].IdelNum {
			return resp.AcceleratorCardSummury[i].Mode < resp.AcceleratorCardSummury[j].Mode
		}
		return resp.AcceleratorCardSummury[i].IdelNum > resp.AcceleratorCardSummury[j].IdelNum
	})
	return resp, nil
}

func (h *ClusterHandler) GetPlatformGPUTop(
	ctx context.Context,
	req *clustersv1alpha1.GetPlatformGPUTopRequest,
) (*clustersv1alpha1.GetPlatformGPUTopResponse, error) {
	clusters, err := h.clusterService.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	temp := map[string]*clustersv1alpha1.GPUSummary{}
	for _, cluster := range clusters {
		if convertCondition2State(cluster.Status) != clustersv1alpha1.ClusterState_RUNNING {
			continue
		}
		// Query all needed metrics and parse to map keyed by uuid.
		memUsedVec, err := h.monitoringService.QueryGPUVector(ctx, cluster.Name, "", "", "", monitoring.GPUQueryTypeMemoryUsed)
		if err != nil {
			return nil, err
		}
		memAllocatedMap, err := h.monitoringService.QueryGPUMap(ctx, cluster.Name, "", "", "", "UUID", monitoring.GPUQueryTypeMemoryAllocated)
		if err != nil {
			return nil, err
		}
		coreUsedMap, err := h.monitoringService.QueryGPUMap(ctx, cluster.Name, "", "", "", "UUID", monitoring.GPUQueryTypeCoreUsed)
		if err != nil {
			return nil, err
		}
		coreAllocatedMap, err := h.monitoringService.QueryGPUMap(ctx, cluster.Name, "", "", "", "UUID", monitoring.GPUQueryTypeCoreAllocated)
		if err != nil {
			return nil, err
		}
		memTotalMap, err := h.monitoringService.QueryGPUMap(ctx, cluster.Name, "", "", "", "UUID", monitoring.GPUQueryTypeMemoryTotal)
		if err != nil {
			return nil, err
		}
		coreTotalMap, err := h.monitoringService.QueryGPUMap(ctx, cluster.Name, "", "", "", "UUID", monitoring.GPUQueryTypeCoreTotal)
		if err != nil {
			return nil, err
		}
		// Summary of metrics by modelname.
		for i := range memUsedVec {
			mode := string(memUsedVec[i].Metric["modelName"])
			uuid := string(memUsedVec[i].Metric["UUID"])
			if _, ok := temp[mode]; !ok {
				temp[mode] = &clustersv1alpha1.GPUSummary{
					Model:         mode,
					Total:         0,
					MemAllocated:  0,
					MemUsage:      0,
					CoreAllocated: 0,
					CoreUsage:     0,
				}
			}

			memTotal, ok1 := memTotalMap[uuid]
			coreTotal, ok2 := coreTotalMap[uuid]
			if !ok1 || !ok2 || memTotal.Value == 0 || coreTotal.Value == 0 {
				continue
			}
			temp[mode].Total += 1
			temp[mode].MemUsage += float64(memUsedVec[i].Value / memTotal.Value)
			if sample, ok := memAllocatedMap[uuid]; ok && sample != nil {
				temp[mode].MemAllocated += float64(sample.Value / memTotal.Value)
			}
			if sample, ok := coreUsedMap[uuid]; ok && sample != nil {
				temp[mode].CoreUsage += float64(sample.Value) / float64(coreTotal.Value) * 100
			}
			if sample, ok := coreAllocatedMap[uuid]; ok && sample != nil {
				temp[mode].CoreAllocated += float64(sample.Value) / float64(coreTotal.Value) * 100
			}
		}
	}
	// Calculate the average of each mode.
	gpus := []*clustersv1alpha1.GPUSummary{}
	for _, model := range temp {
		gpus = append(gpus, &clustersv1alpha1.GPUSummary{
			Model:         model.Model,
			Total:         model.Total,
			MemAllocated:  model.MemAllocated / float64(model.Total),
			MemUsage:      model.MemUsage / float64(model.Total),
			CoreAllocated: model.CoreAllocated / float64(model.Total),
			CoreUsage:     model.CoreUsage / float64(model.Total),
		})
	}
	switch req.GetRankOption() {
	case clustersv1alpha1.RankOption_RANK_OPTION_UNSPECIFIED:
		sort.Slice(gpus, func(i, j int) bool {
			return gpus[i].Total > gpus[j].Total
		})
	case clustersv1alpha1.RankOption_RANK_OPTION_CORE:
		sort.Slice(gpus, func(i, j int) bool {
			return gpus[i].CoreUsage > gpus[j].CoreUsage
		})
	case clustersv1alpha1.RankOption_RANK_OPTION_MEMORY:
		sort.Slice(gpus, func(i, j int) bool {
			return gpus[i].MemUsage > gpus[j].MemUsage
		})
	}
	if req.GetTopn() > 0 && req.GetTopn() < int32(len(gpus)) {
		gpus = gpus[:req.GetTopn()]
	}
	return &clustersv1alpha1.GetPlatformGPUTopResponse{Gpus: gpus}, nil
}

func buildClusterFromRequest(req *clustersv1alpha1.IntegrateClusterRequest) *clustercrdv1alpha1.Cluster {
	return &clustercrdv1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       clustercrdv1alpha1.ClusterResourceKind,
			APIVersion: clustercrdv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   req.GetName(),
			Labels: req.GetLabels(),
			Annotations: map[string]string{
				constants.ClusterAliasAnnotationKey:       req.GetAliasName(),
				constants.ClusterDescriptionAnnotationKey: req.GetDescription(),
			},
		},
		Spec: clustercrdv1alpha1.ClusterSpec{
			Provider:          req.GetProvider().String(),
			Type:              req.GetType().String(),
			SecretRef:         &clustercrdv1alpha1.LocalSecretReference{}, // for the secret of kubeconfig, to be filled.
			PrometheusAddress: req.GetPrometheusAddress(),
			GatewayAddress:    req.GetGatewayAddress(),
		},
	}
}

type clusterMetric struct {
	cpuUsage           float64
	memoryUsage        float64
	gpuCoreUsage       float64
	gpuMemoryUsage     float64
	gpuCoreAllocated   float64
	gpuMemoryAllocated float64
}

func (c *clusterMetric) Save(rt monitoringv1alpha1.ResourceType, usage, allocated prometheusmodel.Vector) {
	var usageNum, allocatedNum float64
	if len(usage) > 0 {
		usageNum = float64(usage[0].Value)
	}
	if len(allocated) > 0 {
		allocatedNum = float64(allocated[0].Value)
	}

	switch rt {
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_UNSPECIFIED:
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_CPU:
		c.cpuUsage = usageNum
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_MEMORY:
		c.memoryUsage = usageNum
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_CORE:
		c.gpuCoreUsage = usageNum
		c.gpuCoreAllocated = allocatedNum
	case monitoringv1alpha1.ResourceType_RESOURCE_TYPE_GPU_MEMORY:
		c.gpuMemoryUsage = usageNum
		c.gpuMemoryAllocated = allocatedNum
	}
}

func (h *ClusterHandler) ListClusterVersions(ctx context.Context, _ *emptypb.Empty) (*clustersv1alpha1.ListClusterVersionsResponse, error) {
	clusters, err := h.clusterService.ListClusters(ctx)
	if err != nil {
		return nil, err
	}
	versions := make(map[string]bool)
	for _, cluster := range clusters {
		versions[cluster.Status.KubernetesVersion] = true
	}
	res := make([]string, 0)
	for k := range versions {
		res = append(res, k)
	}
	return &clustersv1alpha1.ListClusterVersionsResponse{Versions: res}, nil
}

func (h *ClusterHandler) GetClusterPlugins(ctx context.Context, req *clustersv1alpha1.GetClusterPluginsRequest) (*clustersv1alpha1.GetClusterPluginsResponse, error) {
	res := &clustersv1alpha1.GetClusterPluginsResponse{}

	labelSelector := labels.Set{"app.kubernetes.io/component": "hami-scheduler"}.String()
	pods, err := h.coreService.ListPods(ctx, req.GetName(), "", labelSelector)
	if err != nil {
		return nil, err
	}
	if len(pods) > 0 {
		res.Plugins = append(res.Plugins, &clustersv1alpha1.KantaloupePlugin{
			Name:      clustersv1alpha1.KantaloupePluginName_HAMI,
			Namespace: pods[0].GetNamespace(),
		})
	}

	// other plugins....

	return res, nil
}

func (h *ClusterHandler) GetClusterCardRequestType(ctx context.Context, req *clustersv1alpha1.GetClusterCardRequestTypeRequest) (*clustersv1alpha1.GetClusterCardRequestTypeResponse, error) {
	cluster, err := h.clusterService.GetCluster(ctx, req.GetName())
	if err != nil {
		return nil, err
	}

	res := &clustersv1alpha1.GetClusterCardRequestTypeResponse{
		RequestTypes: []*clustersv1alpha1.CardRequestType{},
	}

	// Nvidia cluster.
	if cluster.Spec.Type == clustersv1alpha1.ClusterType_NVIDIA.String() {
		res.RequestTypes = append(res.RequestTypes, &clustersv1alpha1.CardRequestType{
			RequestType: "NVIDIA GPU",
			ResourceNames: []*clustersv1alpha1.ResourceName{
				{
					CardModel:    "",
					ResourceKeys: []string{"nvidia.com/gpu"},
				},
			},
		})
		res.RequestTypes = append(res.RequestTypes, &clustersv1alpha1.CardRequestType{
			RequestType: "NVIDIA vGPU",
			ResourceNames: []*clustersv1alpha1.ResourceName{
				{
					CardModel:    "",
					ResourceKeys: []string{"nvidia.com/gpu", "nvidia.com/gpumem", "nvidia.com/gpucores"},
				},
			},
		})
	}

	// MetaX cluster.
	if cluster.Spec.Type == clustersv1alpha1.ClusterType_METAX.String() {
		res.RequestTypes = append(res.RequestTypes, &clustersv1alpha1.CardRequestType{
			RequestType: "MetaX GPU",
			ResourceNames: []*clustersv1alpha1.ResourceName{
				{
					CardModel:    "",
					ResourceKeys: []string{"metax-tech.com/gpu"},
				},
			},
		})
		res.RequestTypes = append(res.RequestTypes, &clustersv1alpha1.CardRequestType{
			RequestType: "MetaX sGPU",
			ResourceNames: []*clustersv1alpha1.ResourceName{
				{
					CardModel:    "",
					ResourceKeys: []string{"metax-tech.com/sgpu", "metax-tech.com/vmemory", "metax-tech.com/vcore"},
				},
			},
		})
	}

	// Neuron cluster.
	if cluster.Spec.Type == clustersv1alpha1.ClusterType_NEURON.String() {
		res.RequestTypes = append(res.RequestTypes, &clustersv1alpha1.CardRequestType{
			RequestType: "Neuron GPU",
			ResourceNames: []*clustersv1alpha1.ResourceName{
				{
					CardModel:    "",
					ResourceKeys: []string{"aws.amazon.com/neuron"},
				},
			},
		})
		res.RequestTypes = append(res.RequestTypes, &clustersv1alpha1.CardRequestType{
			RequestType: "Neuron Core",
			ResourceNames: []*clustersv1alpha1.ResourceName{
				{
					CardModel:    "",
					ResourceKeys: []string{"aws.amazon.com/neuroncore"},
				},
			},
		})
	}

	// Ascend cluster.
	if cluster.Spec.Type == clustersv1alpha1.ClusterType_ASCEND.String() {
		models, err := h.ListAscendCardModel(ctx, req.GetName())
		if err != nil {
			return nil, err
		}
		npuResourceNames := []*clustersv1alpha1.ResourceName{}
		vgpuResourceNames := []*clustersv1alpha1.ResourceName{}
		for _, model := range models {
			npuResourceNames = append(npuResourceNames, &clustersv1alpha1.ResourceName{
				CardModel:    model,
				ResourceKeys: []string{fmt.Sprintf("huawei.com/%s", model)},
			})
			vgpuResourceNames = append(vgpuResourceNames, &clustersv1alpha1.ResourceName{
				CardModel:    model,
				ResourceKeys: []string{fmt.Sprintf("huawei.com/%s", model), fmt.Sprintf("huawei.com/%s-memory", model)},
			})
		}

		res.RequestTypes = append(res.RequestTypes, &clustersv1alpha1.CardRequestType{
			RequestType:   "ASCEND NPU",
			ResourceNames: npuResourceNames,
		})
		res.RequestTypes = append(res.RequestTypes, &clustersv1alpha1.CardRequestType{
			RequestType:   "ASCEND vNPU",
			ResourceNames: vgpuResourceNames,
		})
	}

	return res, nil
}

func (h *ClusterHandler) ListAscendCardModel(ctx context.Context, cluster string) ([]string, error) {
	nodes, err := h.coreService.ListNodes(ctx, cluster)
	if err != nil {
		return nil, err
	}

	cardModelSet := map[string]struct{}{}
	for _, node := range nodes {
		for key := range node.GetAnnotations() {
			if !strings.HasPrefix(key, "hami.io/node-register-") {
				continue
			}
			model := strings.TrimPrefix(key, "hami.io/node-register-")
			if model != "" {
				cardModelSet[model] = struct{}{}
			}
		}
	}

	cardModels := make([]string, 0, len(cardModelSet))
	for model := range cardModelSet {
		cardModels = append(cardModels, model)
	}
	sort.Strings(cardModels)
	return cardModels, nil
}
