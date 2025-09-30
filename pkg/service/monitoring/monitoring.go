package monitoring

import (
	"context"
	"errors"
	"time"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prometheusmodel "github.com/prometheus/common/model"

	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/monitoring"
)

type QueryType string

const (
	QueryTypeCPUTotal           QueryType = "cpu_total"
	QueryTypeCPUAllocated       QueryType = "cpu_allocated"
	QueryTypeCPUUsed            QueryType = "cpu_used"
	QueryTypeMemoryTotal        QueryType = "mem_total"
	QueryTypeMemoryAllocated    QueryType = "mem_allocated"
	QueryTypeMemoryUsed         QueryType = "mem_used"
	QueryTypeGPUCoreTotal       QueryType = "gpucore_total"
	QueryTypeGPUCoreAllocated   QueryType = "gpucore_allocated"
	QueryTypeGPUCoreUsed        QueryType = "gpucore_used"
	QueryTypeGPUMemoryTotal     QueryType = "gpumem_total"
	QueryTypeGPUMemoryAllocated QueryType = "gpumem_allocated"
	QueryTypeGPUMemoryUsed      QueryType = "gpumem_used"
)

type GPUQueryType string

const (
	GPUQueryTypeCount           GPUQueryType = "count"
	GPUQueryTypeCoreTotal       GPUQueryType = "core_total"
	GPUQueryTypeCoreAllocated   GPUQueryType = "core_allocated"
	GPUQueryTypeCoreUsed        GPUQueryType = "core_used"
	GPUQueryTypeMemoryTotal     GPUQueryType = "mem_total"
	GPUQueryTypeMemoryAllocated GPUQueryType = "mem_allocated"
	GPUQueryTypeMemoryUsed      GPUQueryType = "mem_used"
	GPUQueryTypePower           GPUQueryType = "power"
	GPUQueryTypeTemperature     GPUQueryType = "temp"
	GPUQueryTypeErrors          GPUQueryType = "errors"
)

type Service interface {
	QueryVector(ctx context.Context, query string) (prometheusmodel.Vector, error)
	QueryRange(ctx context.Context, query string, r prometheusv1.Range) (prometheusmodel.Matrix, error)
	// QueryWorkload returns exactly one metric for a workload.
	QueryWorkload(ctx context.Context, cluster, namespace, name string, queryType QueryType) (prometheusmodel.Vector, error)
	QueryWorkloadRange(ctx context.Context, cluster, namespace, name string, queryType QueryType, r prometheusv1.Range) (prometheusmodel.Matrix, error)
	QueryWorkloadVector(ctx context.Context, cluster, node, namespace string, queryType QueryType) (prometheusmodel.Vector, error)
	QueryWorkloadMap(ctx context.Context, cluster, node, namespace, keyLabel string, queryType QueryType) (monitoring.PrometheusVectorMap, error)
	// QueryNode returns exactly one metric for a node.
	QueryNode(ctx context.Context, cluster, node string, queryType QueryType) (prometheusmodel.Vector, error)
	QueryNodeRange(ctx context.Context, cluster, node string, queryType QueryType, r prometheusv1.Range) (prometheusmodel.Matrix, error)
	QueryNodeVector(ctx context.Context, cluster string, queryType QueryType) (prometheusmodel.Vector, error)
	QueryNodeMap(ctx context.Context, cluster, keyLabel string, queryType QueryType) (monitoring.PrometheusVectorMap, error)
	// QueryCluster returns exactly one metric for a cluster.
	QueryCluster(ctx context.Context, cluster string, queryType QueryType) (prometheusmodel.Vector, error)
	QueryClusterRange(ctx context.Context, cluster string, queryType QueryType, r prometheusv1.Range) (prometheusmodel.Matrix, error)
	QueryClusterVector(ctx context.Context, queryType QueryType) (prometheusmodel.Vector, error)
	QueryClusterMap(ctx context.Context, keyLabel string, queryType QueryType) (monitoring.PrometheusVectorMap, error)
	// QueryGPU returns exactly one metrics for given gpu.
	QueryGPU(ctx context.Context, cluster, uuid string, queryType GPUQueryType) (prometheusmodel.Vector, error)
	QueryGPURange(ctx context.Context, cluster, uuid string, queryType GPUQueryType, r prometheusv1.Range) (prometheusmodel.Matrix, error)
	QueryGPUVector(ctx context.Context, cluster, node, vendor, model string, queryType GPUQueryType) (prometheusmodel.Vector, error)
	QueryGPUMap(ctx context.Context, cluster, node, vendor, model, keyLabel string, queryType GPUQueryType) (monitoring.PrometheusVectorMap, error)

	QueryPlatform(ctx context.Context, queryType QueryType) (prometheusmodel.Vector, error)
	QueryPlatformRange(ctx context.Context, queryType QueryType, r prometheusv1.Range) (prometheusmodel.Matrix, error)
}

var _ Service = (*service)(nil)

type service struct {
	client engine.PrometheusInterface
}

// QueryCluster implements Service.
func (s *service) QueryCluster(ctx context.Context, cluster string, queryType QueryType) (prometheusmodel.Vector, error) {
	query := buildClusterQuery(cluster, queryType)
	vec, err := s.QueryVector(ctx, query)
	if err != nil {
		return nil, err
	}
	return vec, assertOneElement(vec, query)
}

// QueryClusterMap implements Service.
func (s *service) QueryClusterMap(ctx context.Context, keyLabel string, queryType QueryType) (monitoring.PrometheusVectorMap, error) {
	query := buildClusterQuery("", queryType)
	vec, err := s.QueryVector(ctx, query)
	if err != nil {
		return nil, err
	}
	return monitoring.VectorToMapByLabel(vec, keyLabel)
}

// QueryClusterRange implements Service.
func (s *service) QueryClusterRange(ctx context.Context, cluster string, queryType QueryType, r prometheusv1.Range) (prometheusmodel.Matrix, error) {
	query := surroundWithSum(buildClusterQuery(cluster, queryType))
	return s.QueryRange(ctx, query, r)
}

// QueryClusterVector implements Service.
func (s *service) QueryClusterVector(ctx context.Context, queryType QueryType) (prometheusmodel.Vector, error) {
	query := buildClusterQuery("", queryType)
	return s.QueryVector(ctx, query)
}

// QueryGPU implements Service.
func (s *service) QueryGPU(ctx context.Context, cluster, uuid string, queryType GPUQueryType) (prometheusmodel.Vector, error) {
	query := buildGPUQuery(cluster, "", "", "", uuid, queryType)
	vec, err := s.QueryVector(ctx, query)
	if err != nil {
		return nil, err
	}
	return vec, assertOneElement(vec, query)
}

// QueryGPUMap implements Service.
func (s *service) QueryGPUMap(ctx context.Context, cluster, node, vendor, model, keyLabel string, queryType GPUQueryType) (monitoring.PrometheusVectorMap, error) {
	query := buildGPUQuery(cluster, node, vendor, model, "", queryType)
	vec, err := s.QueryVector(ctx, query)
	if err != nil {
		return nil, err
	}
	return monitoring.VectorToMapByLabel(vec, keyLabel)
}

// QueryGPURange implements Service.
func (s *service) QueryGPURange(ctx context.Context, cluster, uuid string, queryType GPUQueryType, r prometheusv1.Range) (prometheusmodel.Matrix, error) {
	query := surroundWithSum(buildGPUQuery(cluster, "", "", "", uuid, queryType))
	return s.QueryRange(ctx, query, r)
}

// QueryGPUVector implements Service.
func (s *service) QueryGPUVector(ctx context.Context, cluster, node, vendor, model string, queryType GPUQueryType) (prometheusmodel.Vector, error) {
	query := buildGPUQuery(cluster, node, vendor, model, "", queryType)
	return s.QueryVector(ctx, query)
}

// QueryNode implements Service.
func (s *service) QueryNode(ctx context.Context, cluster, node string, queryType QueryType) (prometheusmodel.Vector, error) {
	query := buildNodeQuery(cluster, node, queryType)
	vec, err := s.QueryVector(ctx, query)
	if err != nil {
		return nil, err
	}
	return vec, assertOneElement(vec, query)
}

// QueryNodeMap implements Service.
func (s *service) QueryNodeMap(ctx context.Context, cluster, keyLabel string, queryType QueryType) (monitoring.PrometheusVectorMap, error) {
	query := buildNodeQuery(cluster, "", queryType)
	vec, err := s.QueryVector(ctx, query)
	if err != nil {
		return nil, err
	}
	return monitoring.VectorToMapByLabel(vec, keyLabel)
}

// QueryNodeRange implements Service.
func (s *service) QueryNodeRange(ctx context.Context, cluster, node string, queryType QueryType, r prometheusv1.Range) (prometheusmodel.Matrix, error) {
	query := surroundWithSum(buildNodeQuery(cluster, node, queryType))
	return s.QueryRange(ctx, query, r)
}

// QueryNodeVector implements Service.
func (s *service) QueryNodeVector(ctx context.Context, cluster string, queryType QueryType) (prometheusmodel.Vector, error) {
	query := buildNodeQuery(cluster, "", queryType)
	return s.QueryVector(ctx, query)
}

// QueryPlatform implements Service.
func (s *service) QueryPlatform(ctx context.Context, queryType QueryType) (prometheusmodel.Vector, error) {
	query := buildPlatformQuery(queryType)
	vec, err := s.QueryVector(ctx, query)
	if err != nil {
		return nil, err
	}
	return vec, assertOneElement(vec, query)
}

// QueryPlatformRange implements Service.
func (s *service) QueryPlatformRange(ctx context.Context, queryType QueryType, r prometheusv1.Range) (prometheusmodel.Matrix, error) {
	query := surroundWithSum(buildPlatformQuery(queryType))
	return s.QueryRange(ctx, query, r)
}

// QueryWorkload implements Service.
func (s *service) QueryWorkload(ctx context.Context, cluster, namespace, name string, queryType QueryType) (prometheusmodel.Vector, error) {
	query := buildWorkLoadQuery(cluster, "", namespace, name, queryType)
	vec, err := s.QueryVector(ctx, query)
	if err != nil {
		return nil, err
	}
	return vec, assertOneElement(vec, query)
}

// QueryWorkloadMap implements Service.
func (s *service) QueryWorkloadMap(ctx context.Context, cluster, node, namespace, keyLabel string, queryType QueryType) (monitoring.PrometheusVectorMap, error) {
	query := buildWorkLoadQuery(cluster, node, namespace, "", queryType)
	vec, err := s.QueryVector(ctx, query)
	if err != nil {
		return nil, err
	}
	return monitoring.VectorToMapByLabel(vec, keyLabel)
}

// QueryWorkloadRange implements Service.
func (s *service) QueryWorkloadRange(ctx context.Context, cluster, namespace, name string, queryType QueryType, r prometheusv1.Range) (prometheusmodel.Matrix, error) {
	query := surroundWithSum(buildWorkLoadQuery(cluster, "", namespace, name, queryType))
	return s.QueryRange(ctx, query, r)
}

// QueryWorkloadVector implements Service.
func (s *service) QueryWorkloadVector(ctx context.Context, cluster, node, namespace string, queryType QueryType) (prometheusmodel.Vector, error) {
	query := buildWorkLoadQuery(cluster, node, namespace, "", queryType)
	return s.QueryVector(ctx, query)
}

// Query implements Service.
func (s *service) QueryVector(ctx context.Context, query string) (prometheusmodel.Vector, error) {
	value, err := s.client.Query(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}
	switch v := value.(type) {
	case prometheusmodel.Vector:
		return v, nil
	default:
		return nil, errors.New("unexpected result type")
	}
}

// QueryRange implements Service.
func (s *service) QueryRange(
	ctx context.Context, query string, r prometheusv1.Range,
) (prometheusmodel.Matrix, error) {
	value, err := s.client.QueryRange(ctx, query, r)
	if err != nil {
		return nil, err
	}
	switch v := value.(type) {
	case prometheusmodel.Matrix:
		return v, nil
	default:
		return nil, errors.New("unexpected result type")
	}
}

func NewService(client engine.PrometheusInterface) Service {
	return &service{
		client: client,
	}
}
