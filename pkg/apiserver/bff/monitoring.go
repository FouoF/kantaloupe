package bff

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prometheusmodel "github.com/prometheus/common/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	monitoringv1alpha1 "github.com/dynamia-ai/kantaloupe/api/monitoring/v1alpha1"
	kantaloupeapi "github.com/dynamia-ai/kantaloupe/api/v1"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/monitoring/resource"
	"github.com/dynamia-ai/kantaloupe/pkg/monitoring/workload"
	"github.com/dynamia-ai/kantaloupe/pkg/service/cluster"
	"github.com/dynamia-ai/kantaloupe/pkg/service/core"
	"github.com/dynamia-ai/kantaloupe/pkg/service/kantaloupeflow"
	"github.com/dynamia-ai/kantaloupe/pkg/service/monitoring"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/defaults"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/errs"
	monitorutils "github.com/dynamia-ai/kantaloupe/pkg/utils/monitoring"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/timeutil"
)

var _ kantaloupeapi.MonitoringServer = &MonitoringHandler{}

type MonitoringHandler struct {
	kantaloupeapi.UnimplementedMonitoringServer
	monitoringService monitoring.Service
	workloadService   core.Service
	clusterService    cluster.Service
	kfService         kantaloupeflow.Service
}

// GetResourceTrend implements v1.MonitoringServer.
func (m *MonitoringHandler) GetResourceTrend(ctx context.Context, req *monitoringv1alpha1.ResourceTrendRequest) (*monitoringv1alpha1.ResourceTrendResponse, error) {
	queries := ConvertResourceType2QueryType(req.GetResourceType())
	timeRange := timeutil.NewTimeRange(req.GetStart(), req.GetEnd(), req.GetRange())
	r := prometheusv1.Range{
		Start: timeRange.Start,
		End:   timeRange.End,
		Step:  timeRange.GetStepAndAlignRange(time.Second * time.Duration(req.GetStep())),
	}
	totalMatrix, err := m.monitoringService.QueryClusterRange(ctx, req.GetCluster(), queries["total"], r)
	if err != nil {
		return nil, err
	}
	usedMatrix, err := m.monitoringService.QueryClusterRange(ctx, req.GetCluster(), queries["used"], r)
	if err != nil {
		return nil, err
	}
	allocatedMatrix, err := m.monitoringService.QueryClusterRange(ctx, req.GetCluster(), queries["allocated"], r)
	if err != nil {
		return nil, err
	}
	return calculateUsage(totalMatrix, allocatedMatrix, usedMatrix, r.Start, r.End, r.Step), nil
}

// GetNodeResourceTrend implements v1.MonitoringServer.
func (m *MonitoringHandler) GetNodeResourceTrend(ctx context.Context, req *monitoringv1alpha1.NodeResourceTrendRequest) (*monitoringv1alpha1.ResourceTrendResponse, error) {
	queries := ConvertResourceType2QueryType(req.GetResourceType())
	timeRange := timeutil.NewTimeRange(req.GetStart(), req.GetEnd(), req.GetRange())
	r := prometheusv1.Range{
		Start: timeRange.Start,
		End:   timeRange.End,
		Step:  timeRange.GetStepAndAlignRange(time.Second * time.Duration(req.GetStep())),
	}
	totalMatrix, err := m.monitoringService.QueryNodeRange(ctx, req.GetCluster(), req.GetNode(), queries["total"], r)
	if err != nil {
		return nil, err
	}
	usedMatrix, err := m.monitoringService.QueryNodeRange(ctx, req.GetCluster(), req.GetNode(), queries["used"], r)
	if err != nil {
		return nil, err
	}
	allocatedMatrix, err := m.monitoringService.QueryNodeRange(ctx, req.GetCluster(), req.GetNode(), queries["allocated"], r)
	if err != nil {
		return nil, err
	}
	return calculateUsage(totalMatrix, allocatedMatrix, usedMatrix, r.Start, r.End, r.Step), nil
}

func (m *MonitoringHandler) GetGpuResourceTrend(ctx context.Context, req *monitoringv1alpha1.GpuResourceTrendRequest) (*monitoringv1alpha1.ResourceTrendResponse, error) {
	queries := ConvertResourceType2GPUQueryType(req.GetResourceType())
	timeRange := timeutil.NewTimeRange(req.GetStart(), req.GetEnd(), req.GetRange())
	r := prometheusv1.Range{
		Start: timeRange.Start,
		End:   timeRange.End,
		Step:  timeRange.GetStepAndAlignRange(time.Second * time.Duration(req.GetStep())),
	}
	if len(queries) == 1 {
		usedMatrix, err := m.monitoringService.QueryGPURange(ctx, req.GetCluster(), req.GetUuid(), queries["used"], r)
		if err != nil {
			return nil, err
		}
		usedMatrix = monitorutils.FillMissingMatrixPoints(usedMatrix, r.Start, r.End, r.Step)
		resp := &monitoringv1alpha1.ResourceTrendResponse{
			Data: []*monitoringv1alpha1.TimeSeries{
				{
					Metric: "used",
					Points: make([]*monitoringv1alpha1.TimeSeriesPoint, 0, len(usedMatrix)),
				},
			},
		}
		if len(usedMatrix) > 0 {
			for _, pair := range usedMatrix[0].Values {
				point := &monitoringv1alpha1.TimeSeriesPoint{
					Timestamp: int64(pair.Timestamp),
					Value:     nil,
				}
				if pair.Value != -1.0 {
					point.Value = wrapperspb.Double(float64(pair.Value))
				}
				resp.Data[0].Points = append(resp.Data[0].Points, point)
			}
		}
		return resp, nil
	}
	totalMatrix, err := m.monitoringService.QueryGPURange(ctx, req.GetCluster(), req.GetUuid(), queries["total"], r)
	if err != nil {
		return nil, err
	}
	usedMatrix, err := m.monitoringService.QueryGPURange(ctx, req.GetCluster(), req.GetUuid(), queries["used"], r)
	if err != nil {
		return nil, err
	}
	allocatedMatrix, err := m.monitoringService.QueryGPURange(ctx, req.GetCluster(), req.GetUuid(), queries["allocated"], r)
	if err != nil {
		return nil, err
	}
	return calculateUsage(totalMatrix, allocatedMatrix, usedMatrix, r.Start, r.End, r.Step), nil
}

func (m *MonitoringHandler) GetKantaloupeflowResourceTrend(ctx context.Context, req *monitoringv1alpha1.KantaloupeflowResourceTrendRequest) (*monitoringv1alpha1.ResourceTrendResponse, error) {
	if req.Cluster == "" || req.Namespace == "" || req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "the cluster, namespace and name can not be empty")
	}

	queries := ConvertResourceType2QueryType(req.GetResourceType())
	timeRange := timeutil.NewTimeRange(req.GetStart(), req.GetEnd(), req.GetRange())
	r := prometheusv1.Range{
		Start: timeRange.Start,
		End:   timeRange.End,
		Step:  timeRange.GetStepAndAlignRange(time.Second * time.Duration(req.GetStep())),
	}
	usedMatrix, err := m.monitoringService.QueryWorkloadRange(ctx, req.GetCluster(), req.GetNamespace(), req.GetName(), queries["used"], r)
	if err != nil {
		return nil, err
	}
	allocatedMatrix, err := m.monitoringService.QueryWorkloadRange(ctx, req.GetCluster(), req.GetNamespace(), req.GetName(), queries["allocated"], r)
	if err != nil {
		return nil, err
	}
	return calculateWorkloadUsage(allocatedMatrix, usedMatrix, r.Start, r.End, r.Step, "used"), nil
}

// GetNodeWorkloadDistribution implements v1.MonitoringServer.
// TODO: Use more reliable method to get node distribution.
func (m *MonitoringHandler) GetNodeWorkloadDistribution(ctx context.Context, req *monitoringv1alpha1.NodeWorkloadDistributionRequest) (*monitoringv1alpha1.WorkloadDistributionResponse, error) {
	// Count the number of workloads per GPU using vGPUPodsDeviceAllocated metric
	query := workload.GetNodeWorkloadQueries(req.Cluster, req.GetNode())

	vector, err := m.monitoringService.QueryVector(ctx, query)
	if err != nil {
		if errors.Is(err, errs.ErrPrometheusClientUninitialized) {
			return &monitoringv1alpha1.WorkloadDistributionResponse{}, nil
		}
		return nil, err
	}

	// Convert vector to NodeGPUWorkloads
	workloads := workload.NodeGPUWorkloads{
		GPUWorkloads: make(map[string]int32, len(vector)),
	}
	for _, sample := range vector {
		workloads.GPUWorkloads[string(sample.Metric["deviceuuid"])] = int32(sample.Value)
	}

	return &monitoringv1alpha1.WorkloadDistributionResponse{
		Data: workload.GetNodeDistribution(workloads),
	}, nil
}

// GetClusterWorkloadDistribution implements v1.MonitoringServer.
func (m *MonitoringHandler) GetClusterWorkloadDistribution(ctx context.Context, req *monitoringv1alpha1.ClusterWorkloadDistributionRequest) (*monitoringv1alpha1.WorkloadDistributionResponse, error) {
	vector, err := m.monitoringService.QueryVector(ctx, fmt.Sprintf(`kantaloupe_workload_gpucore_allocated{cluster="%s"} * 0 + 1`, req.GetCluster()))
	if err != nil {
		if errors.Is(err, errs.ErrPrometheusClientUninitialized) {
			return &monitoringv1alpha1.WorkloadDistributionResponse{}, nil
		}
		return nil, err
	}

	podNames, err := m.getWorkloads(ctx, req.GetCluster())
	if err != nil {
		return nil, err
	}
	workloadVec := filterWorkloads(podNames, vector, `deployment`)

	samples := monitorutils.GroupAndSortVector(workloadVec, `node`, true)

	// Convert vector to ClusterNodeWorkloads
	workloads := workload.ClusterNodeWorkloads{
		NodeWorkloads: make(map[string]int32, len(samples)),
	}
	for _, sample := range samples {
		workloads.NodeWorkloads[string(sample.Metric["node"])] = int32(sample.Value)
	}

	return &monitoringv1alpha1.WorkloadDistributionResponse{
		Data: workload.GetClusterDistribution(workloads),
	}, nil
}

// queryTopNodes handles the common logic for querying top K nodes and building the response.
func (m *MonitoringHandler) queryTopNodes(ctx context.Context, query string) (*monitoringv1alpha1.TopNodeResponse, error) {
	vector, err := m.monitoringService.QueryVector(ctx, query)
	if err != nil {
		if errors.Is(err, errs.ErrPrometheusClientUninitialized) {
			return &monitoringv1alpha1.TopNodeResponse{}, nil
		}
		return nil, err
	}

	// Convert vector to response
	resp := &monitoringv1alpha1.TopNodeResponse{
		Data: make([]*monitoringv1alpha1.DistributionPoint, 0, len(vector)),
	}
	for _, sample := range vector {
		name := string(sample.Metric["node"])
		if name == "" {
			name = string(sample.Metric["nodename"])
		}
		resp.Data = append(resp.Data, &monitoringv1alpha1.DistributionPoint{
			Name:  name,
			Value: int32(sample.Value),
		})
	}

	return resp, nil
}

// GetTopNodes implements v1.MonitoringServer.
func (m *MonitoringHandler) GetTopNodes(ctx context.Context, req *monitoringv1alpha1.TopNodeRequest) (*monitoringv1alpha1.TopNodeResponse, error) {
	limit := defaults.GetTopKLimit(req.GetLimit())
	query, err := resource.GetTopNodesQuery(req.GetResourceType(), req.GetRankingType(), req.GetCluster(), limit)
	if err != nil {
		return nil, err
	}

	return m.queryTopNodes(ctx, query)
}

// GetTopNodeWorkloads implements v1.MonitoringServer.
func (m *MonitoringHandler) GetTopNodeWorkloads(ctx context.Context, req *monitoringv1alpha1.TopNodeWorkloadRequest) (*monitoringv1alpha1.TopNodeResponse, error) {
	vector, err := m.monitoringService.QueryVector(ctx, fmt.Sprintf(`kantaloupe_workload_gpucore_allocated{cluster="%s"} * 0 + 1`, req.GetCluster()))
	if err != nil {
		if errors.Is(err, errs.ErrPrometheusClientUninitialized) {
			return &monitoringv1alpha1.TopNodeResponse{}, nil
		}
		return nil, err
	}

	podNames, err := m.getWorkloads(ctx, req.GetCluster())
	if err != nil {
		return nil, err
	}
	workloadVec := filterWorkloads(podNames, vector, `deployment`)

	samples := monitorutils.GroupAndSortVector(workloadVec, `node`, false)
	limit := defaults.GetTopKLimit(req.GetLimit())
	if len(samples) > int(limit) {
		samples = samples[:limit]
	}

	// Convert vector to response
	resp := &monitoringv1alpha1.TopNodeResponse{
		Data: make([]*monitoringv1alpha1.DistributionPoint, 0, len(vector)),
	}
	for _, sample := range samples {
		resp.Data = append(resp.Data, &monitoringv1alpha1.DistributionPoint{
			Name:  string(sample.Metric["node"]),
			Value: int32(sample.Value),
		})
	}

	return resp, nil
}

func (m *MonitoringHandler) GetKantaloupeflowMemoryDistribution(ctx context.Context, req *monitoringv1alpha1.MemoryDistributionRequest) (*monitoringv1alpha1.MemoryDistributionResponse, error) {
	if req.Cluster == "" || req.Namespace == "" || req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "the cluster, namespace and name can not be empty")
	}

	resp := &monitoringv1alpha1.MemoryDistributionResponse{Data: []*monitoringv1alpha1.DistributionPoint64{}}

	vector, err := m.monitoringService.QueryVector(ctx, fmt.Sprintf(`Device_memory_desc_of_container{cluster="%s",deployment="%s/%s"}`, req.Cluster, req.Namespace, req.Name))
	if err != nil {
		return nil, err
	}
	metricKeys := []string{"context", "data", "module"}
	dataMap := make(map[string]int64)
	for _, key := range metricKeys {
		dataMap[key] = 0
		for _, v := range vector {
			value, err := strconv.ParseInt(string(v.Metric[prometheusmodel.LabelName(key)]), 10, 64)
			if err != nil {
				return nil, err
			}
			dataMap[key] += value
		}
	}
	for k, v := range dataMap {
		resp.Data = append(resp.Data, &monitoringv1alpha1.DistributionPoint64{
			Name:  k,
			Value: v,
		})
	}
	return resp, nil
}

func (m *MonitoringHandler) getWorkloads(ctx context.Context, cluster string) ([]string, error) {
	workloadNames := []string{}
	workloads, err := m.kfService.ListKantaloupeflows(ctx, cluster, corev1.NamespaceAll)
	if err != nil {
		return nil, err
	}
	for _, w := range workloads {
		namespace := w.Namespace
		deployment := fmt.Sprintf("%s/%s", namespace, w.Name)
		workloadNames = append(workloadNames, deployment)
	}
	return workloadNames, nil
}

// filter pods managed by kantaloupeflow from all pods.
//
//nolint:unparam
func filterWorkloads(podNames []string, samples prometheusmodel.Vector, label string) prometheusmodel.Vector {
	res := make(prometheusmodel.Vector, 0)
	nameMap := make(map[string]bool)
	for _, name := range podNames {
		nameMap[name] = true
	}
	for _, sample := range samples {
		if nameMap[string(sample.Metric[prometheusmodel.LabelName(label)])] {
			res = append(res, sample)
		}
	}
	return res
}

func (m *MonitoringHandler) GetCardTopWorkloads(ctx context.Context, req *monitoringv1alpha1.CardTopWorkloadsRequest) (*monitoringv1alpha1.CardTopWorkloadsResponse, error) {
	if req.GetType() == monitoringv1alpha1.RequstType_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "request type must be specified")
	}
	workloads := make([]*monitoringv1alpha1.WorkloadInfo, 0)
	workloadNames, err := m.getWorkloads(ctx, req.GetCluster())
	if err != nil {
		return nil, err
	}
	if req.GetType() == monitoringv1alpha1.RequstType_CORE {
		allocatedVec, err := m.monitoringService.QueryWorkloadVector(ctx, req.GetCluster(), "", "", monitoring.QueryTypeGPUCoreAllocated)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to query core allocated percentage: %v", err)
		}
		usedMap, err := m.monitoringService.QueryWorkloadMap(ctx, req.GetCluster(), "", "", "deployment", monitoring.QueryTypeGPUCoreUsed)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to query core used percentage: %v", err)
		}
		workloadAllocatedVec := filterWorkloads(workloadNames, allocatedVec, "deployment")
		for _, allocated := range workloadAllocatedVec {
			if allocated.Metric["UUID"] != prometheusmodel.LabelValue(req.GetUuid()) {
				continue
			}
			deploymentName := string(allocated.Metric[prometheusmodel.LabelName("deployment")])
			if _, ok := usedMap[deploymentName]; !ok {
				usedMap[deploymentName] = &prometheusmodel.Sample{
					Value: 0,
				}
			}
			workloads = append(workloads, &monitoringv1alpha1.WorkloadInfo{
				Name:          deploymentName,
				CoreAllocated: float64(allocated.Value),
				CoreUsage:     float64(usedMap[deploymentName].Value),
			})
		}
		sort.Slice(workloads, func(i, j int) bool {
			return workloads[i].CoreAllocated > workloads[j].CoreAllocated
		})
		total := len(workloads)
		if req.GetLimit() > 0 && int(req.GetLimit()) < len(workloads) {
			workloads = workloads[:req.GetLimit()]
		}
		return &monitoringv1alpha1.CardTopWorkloadsResponse{Workloads: workloads, Total: int32(total)}, nil
	}
	vec, err := m.monitoringService.QueryGPU(ctx, req.GetCluster(), req.GetUuid(), monitoring.GPUQueryTypeMemoryTotal)
	if err != nil {
		return nil, err
	}
	memoryTotal := vec[0].Value
	allocatedVec, err := m.monitoringService.QueryWorkloadVector(ctx, req.GetCluster(), "", "", monitoring.QueryTypeGPUMemoryAllocated)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query memory allocated percentage: %v", err)
	}
	usedMap, err := m.monitoringService.QueryWorkloadMap(ctx, req.GetCluster(), "", "", "deployment", monitoring.QueryTypeGPUMemoryUsed)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query memory used percentage: %v", err)
	}
	workloadAllocatedVec := filterWorkloads(workloadNames, allocatedVec, "deployment")
	for _, allocated := range workloadAllocatedVec {
		if allocated.Metric["UUID"] != prometheusmodel.LabelValue(req.GetUuid()) {
			continue
		}
		deploymentName := string(allocated.Metric[prometheusmodel.LabelName("deployment")])
		if memoryTotal <= 0 {
			workloads = append(workloads, &monitoringv1alpha1.WorkloadInfo{
				Name:            deploymentName,
				MemoryAllocated: 0,
				MemoryUsage:     0,
			})
			continue
		}
		if _, ok := usedMap[deploymentName]; !ok {
			usedMap[deploymentName] = &prometheusmodel.Sample{
				Value: 0,
			}
		}
		workloads = append(workloads, &monitoringv1alpha1.WorkloadInfo{
			Name:            deploymentName,
			MemoryAllocated: float64(allocated.Value) / float64(memoryTotal) * 100,
			MemoryUsage:     float64(usedMap[deploymentName].Value) / float64(memoryTotal) * 100,
		})
	}
	sort.Slice(workloads, func(i, j int) bool {
		return workloads[i].MemoryAllocated > workloads[j].MemoryAllocated
	})
	total := len(workloads)
	if req.GetLimit() > 0 && int(req.GetLimit()) < len(workloads) {
		workloads = workloads[:req.GetLimit()]
	}
	return &monitoringv1alpha1.CardTopWorkloadsResponse{Workloads: workloads, Total: int32(total)}, nil
}

func RestoreOriginalName(generatedName string) string {
	parts := strings.Split(generatedName, "-")
	if len(parts) <= 2 {
		return generatedName
	}
	return strings.Join(parts[:len(parts)-2], "-")
}

func NewMonitoringHandler(clientManager engine.ClientManagerInterface, prometheus engine.PrometheusInterface) *MonitoringHandler {
	return &MonitoringHandler{
		monitoringService: monitoring.NewService(prometheus),
		workloadService:   core.NewService(clientManager),
		clusterService:    cluster.NewService(clientManager),
		kfService:         kantaloupeflow.NewService(clientManager),
	}
}

func (m *MonitoringHandler) GetClusterWorkloadsTop(ctx context.Context, req *monitoringv1alpha1.GetClusterWorkloadsTopRequest) (*monitoringv1alpha1.GetClusterWorkloadsTopResponse, error) {
	timeRange := timeutil.NewTimeRange(0, 0, req.GetRange())
	r := prometheusv1.Range{
		Start: timeRange.Start,
		End:   timeRange.End,
		Step:  timeRange.GetStepAndAlignRange(time.Second * time.Duration(req.GetStep())),
	}
	switch req.GetType() {
	case monitoringv1alpha1.RequstType_CORE:
		vec, err := m.getTopnWorkloads(ctx, req.GetCluster(), fmt.Sprintf(`avg_over_time(sum by (deployment) (kantaloupe_workload_gpucore_used{cluster="%s"})[%s:])`, req.GetCluster(), req.GetRange()), req.GetLimit())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to query core used percentage: %v", err)
		}
		ret := &monitoringv1alpha1.GetClusterWorkloadsTopResponse{}
		for _, v := range vec {
			name := string(v.Metric[prometheusmodel.LabelName("deployment")])
			values := strings.Split(name, "/")
			if len(values) != 2 {
				klog.InfoS("invalid workload name", "name", name)
				continue
			}
			matrix, err := m.monitoringService.QueryWorkloadRange(ctx, req.GetCluster(), values[0], values[1], monitoring.QueryTypeGPUCoreUsed, r)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to query workload core used percentage trend: %v", err)
			}
			ret.Data = append(ret.Data, convertMatrix2TimeSeries(matrix, name, r.Start, r.End, r.Step))
		}
		return ret, nil
	case monitoringv1alpha1.RequstType_MEMORY:
		vec, err := m.getTopnWorkloads(ctx, req.GetCluster(), fmt.Sprintf(`avg_over_time(sum by (deployment) (kantaloupe_workload_gpumem_used{cluster="%s"})[%s:]) 
		/ avg_over_time(sum by (deployment) (kantaloupe_workload_gpumem_allocated{cluster="%s"})[%s:])`, req.GetCluster(), req.GetRange(), req.GetCluster(), req.GetRange()), req.GetLimit())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to query memory used percentage: %v", err)
		}
		ret := &monitoringv1alpha1.GetClusterWorkloadsTopResponse{}
		for _, v := range vec {
			name := string(v.Metric[prometheusmodel.LabelName("deployment")])
			values := strings.Split(name, "/")
			if len(values) != 2 {
				klog.InfoS("invalid workload name", "name", name)
				continue
			}
			usedMatrix, err := m.monitoringService.QueryWorkloadRange(ctx, req.GetCluster(), values[0], values[1], monitoring.QueryTypeGPUMemoryUsed, r)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to query workload memory used percentage trend: %v", err)
			}
			allocatedMatrix, err := m.monitoringService.QueryWorkloadRange(ctx, req.GetCluster(), values[0], values[1], monitoring.QueryTypeGPUMemoryAllocated, r)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to query workload memory allocated percentage trend: %v", err)
			}
			ret.Data = append(ret.Data, calculateWorkloadUsage(allocatedMatrix, usedMatrix, r.Start, r.End, r.Step, name).Data...)
		}
		return ret, nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid request type: %s", req.GetType())
	}
}

func (m *MonitoringHandler) getTopnWorkloads(ctx context.Context, cluster, query string, n int32) (prometheusmodel.Vector, error) {
	vec, err := m.monitoringService.QueryVector(ctx, query)
	if err != nil {
		return nil, err
	}

	workloadNames, err := m.getWorkloads(ctx, cluster)
	if err != nil {
		return nil, err
	}

	ret := filterWorkloads(workloadNames, vec, "deployment")
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Value > ret[j].Value
	})

	// The first element is the total, so we skip it
	if n > 0 && int(n) < len(ret)-1 {
		ret = ret[1 : n+1]
	}
	return ret, nil
}

func convertMatrix2TimeSeries(matrix prometheusmodel.Matrix, name string, start, end time.Time, step time.Duration) *monitoringv1alpha1.TimeSeries {
	matrix = monitorutils.FillMissingMatrixPoints(matrix, start, end, step)
	Data := monitoringv1alpha1.TimeSeries{
		Metric: name,
		Points: make([]*monitoringv1alpha1.TimeSeriesPoint, 0),
	}
	if len(matrix) > 0 {
		for _, pair := range matrix[0].Values {
			point := &monitoringv1alpha1.TimeSeriesPoint{
				Timestamp: int64(pair.Timestamp),
				Value:     nil,
			}
			if pair.Value != -1.0 {
				point.Value = wrapperspb.Double(float64(pair.Value))
			}
			Data.Points = append(Data.Points, point)
		}
	}
	return &Data
}

func calculateUsage(total, allocated, used prometheusmodel.Matrix, start, end time.Time, step time.Duration) *monitoringv1alpha1.ResourceTrendResponse {
	total = monitorutils.FillMissingMatrixPoints(total, start, end, step)
	allocated = monitorutils.FillMissingMatrixPoints(allocated, start, end, step)
	used = monitorutils.FillMissingMatrixPoints(used, start, end, step)
	resp := &monitoringv1alpha1.ResourceTrendResponse{
		Data: []*monitoringv1alpha1.TimeSeries{
			{
				Metric: "allocated",
				Points: make([]*monitoringv1alpha1.TimeSeriesPoint, 0),
			},
			{
				Metric: "used",
				Points: make([]*monitoringv1alpha1.TimeSeriesPoint, 0),
			},
		},
	}
	if len(total) == 0 {
		return resp
	}

	if len(allocated) > 0 {
		for i := range allocated[0].Values {
			point := &monitoringv1alpha1.TimeSeriesPoint{
				Timestamp: int64(allocated[0].Values[i].Timestamp),
				Value:     nil,
			}
			// If the denominator exists and numerator not exist, the value is 0.
			if total[0].Values[i].Value != -1.0 {
				if allocated[0].Values[i].Value == -1.0 {
					point.Value = wrapperspb.Double(0)
				} else {
					point.Value = wrapperspb.Double(float64(allocated[0].Values[i].Value) / float64(total[0].Values[i].Value) * 100)
				}
			}
			resp.Data[0].Points = append(resp.Data[0].Points, point)
		}
	}
	if len(used) > 0 {
		for i := range used[0].Values {
			point := &monitoringv1alpha1.TimeSeriesPoint{
				Timestamp: int64(used[0].Values[i].Timestamp),
				Value:     nil,
			}
			if total[0].Values[i].Value != -1.0 {
				if used[0].Values[i].Value == -1.0 {
					point.Value = wrapperspb.Double(0)
				} else {
					point.Value = wrapperspb.Double(float64(used[0].Values[i].Value) / float64(total[0].Values[i].Value) * 100)
				}
			}
			resp.Data[1].Points = append(resp.Data[1].Points, point)
		}
	}

	return resp
}

func calculateWorkloadUsage(allocated, used prometheusmodel.Matrix, start, end time.Time, step time.Duration, metricName string) *monitoringv1alpha1.ResourceTrendResponse {
	allocated = monitorutils.FillMissingMatrixPoints(allocated, start, end, step)
	used = monitorutils.FillMissingMatrixPoints(used, start, end, step)
	resp := &monitoringv1alpha1.ResourceTrendResponse{
		Data: []*monitoringv1alpha1.TimeSeries{
			{
				Metric: metricName,
				Points: make([]*monitoringv1alpha1.TimeSeriesPoint, 0),
			},
		},
	}
	if len(allocated) == 0 {
		return resp
	}

	if len(used) > 0 {
		for i := range used[0].Values {
			point := &monitoringv1alpha1.TimeSeriesPoint{
				Timestamp: int64(used[0].Values[i].Timestamp),
				Value:     nil,
			}
			if allocated[0].Values[i].Value != -1.0 {
				if used[0].Values[i].Value == -1.0 {
					point.Value = wrapperspb.Double(0)
				} else {
					if allocated[0].Values[i].Value != 0 {
						point.Value = wrapperspb.Double(float64(used[0].Values[i].Value) / float64(allocated[0].Values[i].Value) * 100)
					} else {
						point.Value = wrapperspb.Double(float64(used[0].Values[i].Value) * 100)
					}
				}
			}
			resp.Data[0].Points = append(resp.Data[0].Points, point)
		}
	}

	return resp
}
