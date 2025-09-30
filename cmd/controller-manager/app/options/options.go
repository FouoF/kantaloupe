package options

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	cliflag "k8s.io/component-base/cli/flag"
	componentbaseconfig "k8s.io/component-base/config"
	"k8s.io/klog/v2"

	namespaceutil "github.com/dynamia-ai/kantaloupe/pkg/utils/namespace"
)

// Options contains everything necessary to create and run controller-manager.
type Options struct {
	// Controllers is the list of controllers to enable or disable
	// '*' means "all enabled by default controllers"
	// 'foo' means "enable 'foo'"
	// '-foo' means "disable 'foo'"
	// first item for a particular name wins
	Controllers []string
	// MultiControllers is the list of member cluster controllers to enable or disable
	// '*' means "all enabled by default controllers"
	// 'foo' means "enable 'foo'"
	// '-foo' means "disable 'foo'"
	// first item for a particular name wins
	MultiControllers           []string
	DefaultDisabledControllers []string
	// LeaderElection defines the configuration of leader election client.
	LeaderElection componentbaseconfig.LeaderElectionConfiguration
	// HealthProbeBindAddress is the TCP address that the controller should bind to
	// for serving health probes
	// It can be set to "0" to disable serving the health probe.
	// Defaults to ":10357".
	HealthProbeBindAddress string
	// ClusterSuccessThreshold is the duration of successes for the cluster to be considered healthy after recovery.
	ClusterSuccessThreshold metav1.Duration
	// ClusterFailureThreshold is the duration of failure for the cluster to be considered unhealthy.
	ClusterFailureThreshold metav1.Duration
	// ClusterStatusUpdateFrequency is the frequency that controller computes and report cluster status.
	// It must work with ClusterMonitorGracePeriod(--cluster-monitor-grace-period) in karmada-controller-manager.
	ClusterStatusUpdateFrequency metav1.Duration
	// MetricsBindAddress is the TCP address that the controller should bind to
	// for serving prometheus metrics.
	// It can be set to "0" to disable the metrics serving.
	// Defaults to ":8080".
	MetricsBindAddress string
	// ConcurrentWorkSyncs is the number of Work objects that are
	// allowed to sync concurrently.
	ConcurrentWorkSyncs int
	// ClusterAPIQPS is the QPS to use while talking with cluster kube-apiserver.
	ClusterAPIQPS float32
	// ClusterAPIBurst is the burst to allow while talking with cluster kube-apiserver.
	ClusterAPIBurst int
	// Controllermanger in debug mode will using out of cluster kube-apiserver.
	DebugMode bool
}

// NewOptions builds an empty options.
func NewOptions() *Options {
	return &Options{
		LeaderElection: componentbaseconfig.LeaderElectionConfiguration{
			LeaderElect:       true,
			ResourceLock:      resourcelock.LeasesResourceLock,
			ResourceNamespace: namespaceutil.GetCurrentNamespaceOrDefault(),
			ResourceName:      "kantaloupe-controller-manager",
		},
	}
}

// AddFlags adds flags to the specified FlagSet.
func (o *Options) AddFlags(flags *pflag.FlagSet, allControllers, allMultiControllers, disabledByDefaultControllers []string) {
	flags.StringSliceVar(&o.Controllers,
		"controllers", []string{"*"}, fmt.Sprintf(
			"A list of controllers to enable. '*' enables all on-by-default controllers, "+
				"'foo' enables the controller named 'foo', '-foo' disables the controller named 'foo'."+
				" \nAll controllers: %s.\nDisabled-by-default controllers: %s.",
			strings.Join(allControllers, ", "), strings.Join(disabledByDefaultControllers, ", "),
		))
	flags.StringSliceVar(&o.MultiControllers, "multi-controllers", []string{"-cleanupInactiveWorkloadController", "*"}, fmt.Sprintf(
		"A list of controllers to enable. '*' enables all on-by-default controllers,"+
			"'foo' enables the controller named 'foo', '-foo' disables the controller named 'foo'. \nAll controllers: %s.\n",
		strings.Join(allMultiControllers, ", ")))
	flags.BoolVar(&o.LeaderElection.LeaderElect,
		"leader-elect", true,
		"Start a leader election client and gain leadership before executing the main loop."+
			" Enable this when running replicated components for high availability.")
	flags.DurationVar(&o.LeaderElection.LeaseDuration.Duration, "leader-elect-lease-duration", 15*time.Second,
		"LeaseDuration is the duration that non-leader candidates will wait to force acquire leadership. "+
			"This is measured against time of last observed ack. Default is 15 seconds.")
	flags.DurationVar(&o.LeaderElection.RenewDeadline.Duration,
		"leader-elect-renew-deadline", 10*time.Second,
		"RenewDeadline is the duration that the acting controlplane will"+
			" retry refreshing leadership before giving up. Default is 10 seconds.")
	flags.DurationVar(&o.LeaderElection.RetryPeriod.Duration,
		"leader-elect-retry-period", 2*time.Second,
		"RetryPeriod is the duration the LeaderElector clients should wait between tries of actions. Default is 2 seconds.")
	flags.StringVar(&o.LeaderElection.ResourceNamespace,
		"leader-elect-resource-namespace", namespaceutil.GetCurrentNamespaceOrDefault(),
		"The namespace of resource object that is used for locking during leader election.")
	flags.Float32Var(&o.ClusterAPIQPS,
		"cluster-api-qps", 40.0, "QPS to use while talking with cluster kube-apiserver.")
	flags.IntVar(&o.ClusterAPIBurst,
		"cluster-api-burst", 60, "Burst to use while talking with cluster kube-apiserver.")
	flags.DurationVar(&o.ClusterSuccessThreshold.Duration,
		"cluster-success-threshold", 30*time.Second,
		"The duration of successes for the cluster to be considered healthy after recovery.")
	flags.DurationVar(&o.ClusterFailureThreshold.Duration,
		"cluster-failure-threshold", 30*time.Second,
		"The duration of failure for the cluster to be considered unhealthy.")
	flags.IntVar(&o.ConcurrentWorkSyncs,
		"concurrent-work-syncs", 5,
		"The number of Works that are allowed to sync concurrently.")
	flags.StringVar(&o.HealthProbeBindAddress,
		"health-probe-bind-address", ":10357",
		"The TCP address that the controller should bind to for serving health probes(e.g. 127.0.0.1:10357, :10357)."+
			"It can be set to \"0\" to disable serving the health probe. Defaults to 0.0.0.0:10357.")
	flags.DurationVar(&o.ClusterStatusUpdateFrequency.Duration,
		"cluster-status-update-frequency", 10*time.Second,
		"Specifies how often karmada-controller-manager posts cluster status to karmada-apiserver.")
	flags.BoolVar(&o.DebugMode, "debug-mode", false,
		"Debug mode will using out of cluster kube-apiserver.")
}

func (o *Options) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}
	fs := fss.FlagSet("generic")
	o.AddFlags(fs, o.Controllers, o.MultiControllers, o.DefaultDisabledControllers)

	fs = fss.FlagSet("klog")
	local := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = strings.Replace(fl.Name, "_", "-", -1)
		fs.AddGoFlag(fl)
	})
	return fss
}
