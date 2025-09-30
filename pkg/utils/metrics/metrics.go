package metrics

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	apimachineryversion "k8s.io/apimachinery/pkg/version"
	compbasemetrics "k8s.io/component-base/metrics"

	clustercrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/helper"
	"github.com/dynamia-ai/kantaloupe/pkg/version"
)

var (
	Defaults        DefaultMetrics
	defaultRegistry compbasemetrics.KubeRegistry
	// MustRegister registers registerable metrics but uses the defaultRegistry,
	// panic upon the first registration that causes an error.
	MustRegister func(...compbasemetrics.Registerable)
	// Register registers a collectable metric but uses the defaultRegistry.
	Register func(compbasemetrics.Registerable) error

	RawMustRegister func(...prometheus.Collector)
)

func init() {
	compbasemetrics.BuildVersion = versionGet

	defaultRegistry = compbasemetrics.NewKubeRegistry()
	MustRegister = defaultRegistry.MustRegister
	Register = defaultRegistry.Register
	RawMustRegister = defaultRegistry.RawMustRegister
}

// DefaultMetrics installs the default prometheus metrics handler.
type DefaultMetrics struct{}

// Install adds the DefaultMetrics handler.
func (m DefaultMetrics) Install(router *mux.Router) {
	RawMustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	RawMustRegister(collectors.NewGoCollector())

	router.Handle("/kapis/metrics", Handler())
}

// Overwrite version.Get.
func versionGet() apimachineryversion.Info {
	info := version.Get()
	return apimachineryversion.Info{
		GitVersion:   info.GitVersion,
		GitCommit:    info.GitCommit,
		GitTreeState: info.GitTreeState,
		BuildDate:    info.BuildDate,
		GoVersion:    info.GoVersion,
		Compiler:     info.Compiler,
		Platform:     info.Platform,
	}
}

// Handler returns an HTTP handler for the DefaultGatherer. It is
// already instrumented with InstrumentHandler (using "prometheus" as handler
// name).
func Handler() http.Handler {
	return promhttp.InstrumentMetricHandler(prometheus.NewRegistry(), promhttp.HandlerFor(defaultRegistry, promhttp.HandlerOpts{}))
}

// RecordClusterStatus records the status of the given cluster.
func RecordClusterStatus(cluster *clustercrdv1alpha1.Cluster) {
	clusterReadyGauge.WithLabelValues(cluster.Name).Set(func() float64 {
		if helper.IsClusterReady(&cluster.Status) {
			return 1
		}
		return 0
	}())

	if cluster.Status.NodeSummary != nil {
		clusterTotalNodeNumberGauge.WithLabelValues(cluster.Name).Set(float64(cluster.Status.NodeSummary.TotalNum))
		clusterReadyNodeNumberGauge.WithLabelValues(cluster.Name).Set(float64(cluster.Status.NodeSummary.ReadyNum))
	}
}

// RecordClusterSyncStatusDuration records the duration of the given cluster syncing status.
func RecordClusterSyncStatusDuration(cluster *clustercrdv1alpha1.Cluster, startTime time.Time) {
	clusterSyncStatusDuration.WithLabelValues(cluster.Name).Observe(time.Since(startTime).Seconds())
}

// RecordClusterSyncStatusCount records the counts of the given cluster syncing status.
func RecordClusterSyncStatusCount(cluster *clustercrdv1alpha1.Cluster) {
	clusterSyncStatusCount.WithLabelValues(cluster.Name).Inc()
}

// RecordClusterStatusControllerReconcileDuration records the duration of the given cluster syncing status.
func RecordClusterStatusControllerReconcileDuration(cluster string, startTime time.Time) {
	clusterStatusControllerReconcileDuration.WithLabelValues(cluster).Observe(time.Since(startTime).Seconds())
}

// RecordClusterStatusControllerReconcileCount records the counts of the given cluster syncing status.
func RecordClusterStatusControllerReconcileCount(cluster string) {
	clusterStatusControllerReconcileCount.WithLabelValues(cluster).Inc()
}

const (
	clusterReadyMetricsName                             = "cluster_ready_state"
	clusterTotalNodeNumberMetricsName                   = "cluster_node_number"
	clusterReadyNodeNumberMetricsName                   = "cluster_ready_node_number"
	clusterMemoryAllocatableMetricsName                 = "cluster_memory_allocatable_bytes"
	clusterCPUAllocatableMetricsName                    = "cluster_cpu_allocatable_number"
	clusterPodAllocatableMetricsName                    = "cluster_pod_allocatable_number"
	clusterMemoryAllocatedMetricsName                   = "cluster_memory_allocated_bytes"
	clusterCPUAllocatedMetricsName                      = "cluster_cpu_allocated_number"
	clusterPodAllocatedMetricsName                      = "cluster_pod_allocated_number"
	clusterSyncStatusDurationMetricsName                = "cluster_sync_status_duration_seconds"
	clusterSyncStatusCountName                          = "cluster_sync_status_count_total"
	clusterStatusControllerReconcileDurationMetricsName = "cluster_sync_status_reconcile_duration_seconds"
	clusterStatusControllerReconcileCountName           = "cluster_sync_status_reconcile_count_total"
)

var (
	// clusterReadyGauge reports if the cluster is ready.
	clusterReadyGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: clusterReadyMetricsName,
		Help: "State of the cluster(1 if ready, 0 otherwise).",
	}, []string{"cluster_name"})

	// clusterTotalNodeNumberGauge reports the number of nodes in the given cluster.
	clusterTotalNodeNumberGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: clusterTotalNodeNumberMetricsName,
		Help: "Number of nodes in the cluster.",
	}, []string{"cluster_name"})

	// clusterReadyNodeNumberGauge reports the number of ready nodes in the given cluster.
	clusterReadyNodeNumberGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: clusterReadyNodeNumberMetricsName,
		Help: "Number of ready nodes in the cluster.",
	}, []string{"cluster_name"})

	// clusterMemoryAllocatableGauge reports the allocatable memory in the given cluster.
	clusterMemoryAllocatableGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: clusterMemoryAllocatableMetricsName,
		Help: "Allocatable cluster memory resource in bytes.",
	}, []string{"cluster_name"})

	// clusterCPUAllocatableGauge reports the allocatable CPU in the given cluster.
	clusterCPUAllocatableGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: clusterCPUAllocatableMetricsName,
		Help: "Number of allocatable CPU in the cluster.",
	}, []string{"cluster_name"})

	// clusterPodAllocatableGauge reports the allocatable Pod number in the given cluster.
	clusterPodAllocatableGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: clusterPodAllocatableMetricsName,
		Help: "Number of allocatable pods in the cluster.",
	}, []string{"cluster_name"})

	// clusterMemoryAllocatedGauge reports the allocated memory in the given cluster.
	clusterMemoryAllocatedGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: clusterMemoryAllocatedMetricsName,
		Help: "Allocated cluster memory resource in bytes.",
	}, []string{"cluster_name"})

	// clusterCPUAllocatedGauge reports the allocated CPU in the given cluster.
	clusterCPUAllocatedGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: clusterCPUAllocatedMetricsName,
		Help: "Number of allocated CPU in the cluster.",
	}, []string{"cluster_name"})

	// clusterPodAllocatedGauge reports the allocated Pod number in the given cluster.
	clusterPodAllocatedGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: clusterPodAllocatedMetricsName,
		Help: "Number of allocated pods in the cluster.",
	}, []string{"cluster_name"})

	// clusterSyncStatusDuration reports the duration of the given cluster syncing status.
	clusterSyncStatusDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: clusterSyncStatusDurationMetricsName,
		Help: "Duration in seconds for syncing the status of the cluster once.",
	}, []string{"cluster_name"})

	// clusterSyncStatusCount reports the count of the given cluster.
	clusterSyncStatusCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: clusterSyncStatusCountName,
		Help: "Number of times cluster status is synchronized",
	}, []string{"cluster_name"})

	// clusterStatusControllerReconcileDuration reports the duration of the cluster_status_controller reconcile cluster.
	clusterStatusControllerReconcileDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: clusterStatusControllerReconcileDurationMetricsName,
		Help: "Duration in seconds for cluster_status_controller reconcile a cluster.",
	}, []string{"cluster_name"})

	// clusterStatusControllerReconcileCount reports the count of the cluster_status_controller reconcile cluster.
	clusterStatusControllerReconcileCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: clusterStatusControllerReconcileCountName,
		Help: "Number of times the cluster_status_controller reconcile a cluster.",
	}, []string{"cluster_name"})
)

// ClusterCollectors returns the collectors about clusters.
func ClusterCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		clusterReadyGauge,
		clusterTotalNodeNumberGauge,
		clusterReadyNodeNumberGauge,
		clusterMemoryAllocatableGauge,
		clusterCPUAllocatableGauge,
		clusterPodAllocatableGauge,
		clusterMemoryAllocatedGauge,
		clusterCPUAllocatedGauge,
		clusterPodAllocatedGauge,
		clusterSyncStatusDuration,
		clusterSyncStatusCount,
		clusterStatusControllerReconcileDuration,
		clusterStatusControllerReconcileCount,
	}
}
