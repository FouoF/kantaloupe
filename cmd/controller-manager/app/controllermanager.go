package app

import (
	"context"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/workqueue"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	clustercrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/cmd/controller-manager/app/options"
	"github.com/dynamia-ai/kantaloupe/pkg/controllers/cluster"
	controllerscontext "github.com/dynamia-ai/kantaloupe/pkg/controllers/context"
	multicontrollers "github.com/dynamia-ai/kantaloupe/pkg/controllers/multicontrollers/controllermanager"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/gclient"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/informermanager"
	"github.com/dynamia-ai/kantaloupe/pkg/version"
)

// NewControllerManagerCommand creates a *cobra.Command object with default parameters.
func NewControllerManagerCommand(ctx context.Context) *cobra.Command {
	opts := options.NewOptions()
	cmd := &cobra.Command{
		Use:  "kantaloupe-controller-manager",
		Long: `The kantaloupe controller manager runs a bunch of controllers`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cliflag.PrintFlags(cmd.Flags())
			// validate options
			if errs := opts.Validate(); len(errs) != 0 {
				return errs.ToAggregate()
			}
			return Run(ctx, opts)
		},
	}
	cmd.SetContext(ctx)
	opts.Controllers = controllers.ControllerNames()
	opts.MultiControllers = multicontrollers.RegisteredMultiControllers.ControllerNames()
	opts.DefaultDisabledControllers = sets.List(controllersDisabledByDefault)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of kantaloupe-controller-manager",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println(version.Get())
		},
	}
	fs := cmd.Flags()
	namedFlagSets := opts.Flags()
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}
	cmd.AddCommand(versionCmd)
	return cmd
}

// Run runs the controller-manager with options. This should never exit.
func Run(ctx context.Context, opts *options.Options) error {
	klog.InfoS("Starting kantaloupe-controller-manager", "version", version.Get())
	config := controllerruntime.GetConfigOrDie()
	config.QPS, config.Burst = opts.ClusterAPIQPS, opts.ClusterAPIBurst
	controllerOptions := controllerruntime.Options{
		Scheme:                     gclient.NewSchema(),
		LeaderElection:             opts.LeaderElection.LeaderElect,
		LeaseDuration:              &opts.LeaderElection.LeaseDuration.Duration,
		RenewDeadline:              &opts.LeaderElection.RenewDeadline.Duration,
		RetryPeriod:                &opts.LeaderElection.RetryPeriod.Duration,
		LeaderElectionID:           opts.LeaderElection.ResourceName,
		LeaderElectionNamespace:    opts.LeaderElection.ResourceNamespace,
		LeaderElectionResourceLock: opts.LeaderElection.ResourceLock,
		HealthProbeBindAddress:     opts.HealthProbeBindAddress,
		LivenessEndpointName:       "/healthz",
		ReadinessEndpointName:      "/readyz",
	}

	controllerManager, err := controllerruntime.NewManager(config, controllerOptions)
	if err != nil {
		klog.ErrorS(err, "Failed to build controller manager")
		return err
	}

	if err := controllerManager.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		klog.ErrorS(err, "Failed to add health check endpoint")
		return err
	}

	if err := controllerManager.AddHealthzCheck("readyz", healthz.Ping); err != nil {
		klog.ErrorS(err, "Failed to add readyz check endpoint")
		return err
	}
	setupControllers(ctx, controllerManager, opts)

	// blocks until the context is done.
	if err := controllerManager.Start(ctx); err != nil {
		klog.ErrorS(err, "Starting kantaloupe subcluster controller manager exits unexpectedly")
		return err
	}

	return nil
}

var controllers = make(controllerscontext.Initializers)

// controllersDisabledByDefault is the set of controllers which is disabled by default.
var controllersDisabledByDefault = sets.New[string]()

func init() {
	controllers["cluster"] = startClusterController
	controllers["multiClusters"] = startMultiClustersController
}

func startClusterController(ctx controllerscontext.Context) (bool, error) {
	opts := ctx.Opts

	clusterPredicateFunc := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			obj := createEvent.Object.(*clustercrdv1alpha1.Cluster)
			return obj.Spec.SecretRef != nil
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			obj := updateEvent.ObjectNew.(*clustercrdv1alpha1.Cluster)
			return obj.Spec.SecretRef != nil
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			obj := deleteEvent.Object.(*clustercrdv1alpha1.Cluster)
			return obj.Spec.SecretRef != nil
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}

	controller := &cluster.Controller{
		Client:                       ctx.Mgr.GetClient(),
		PredicateFunc:                clusterPredicateFunc,
		EventRecorder:                ctx.Mgr.GetEventRecorderFor(cluster.ClusterControllerName),
		InformerManager:              informermanager.GetInstance(),
		ClusterClientSetFunc:         utils.NewClusterClientSet,
		ClusterClientOption:          &utils.ClientOption{QPS: opts.ClusterAPIQPS, Burst: opts.ClusterAPIBurst},
		ClusterStatusUpdateFrequency: opts.ClusterStatusUpdateFrequency,
		ConcurrentWorkSyncs:          opts.ConcurrentWorkSyncs,
		ClusterSuccessThreshold:      opts.ClusterSuccessThreshold,
		ClusterFailureThreshold:      opts.ClusterFailureThreshold,
		ClusterDebugMode:             opts.DebugMode,
	}
	if err := controller.SetupWithManager(ctx.Mgr); err != nil {
		return false, err
	}
	return true, nil
}

func startMultiClustersController(ctx controllerscontext.Context) (bool, error) {
	clusterController := &multicontrollers.Controller{
		Client:              ctx.Mgr.GetClient(),
		EventRecorder:       ctx.Mgr.GetEventRecorderFor(multicontrollers.ControllerName),
		ClusterKubeconfig:   utils.ClusterKubeconfig,
		ClusterClientOption: &utils.ClientOption{QPS: ctx.Opts.ClusterAPIQPS, Burst: ctx.Opts.ClusterAPIBurst},
		Queue:               workqueue.NewNamedRateLimitingQueue(workqueue.NewWithMaxWaitRateLimiter(workqueue.DefaultControllerRateLimiter(), 10*time.Second), multicontrollers.ControllerName),
		ControllerManager:   sync.Map{},
		ConcurrentWorkSyncs: ctx.Opts.ConcurrentWorkSyncs,
		MultiControllers:    ctx.Opts.MultiControllers,
	}
	if err := clusterController.SetupWithManager(ctx.Mgr); err != nil {
		klog.ErrorS(err, "Failed to setup multi clusters controller")
		return false, err
	}
	return true, nil
}

// setupControllers initialize controllers and setup one by one.
// Note: ignore cyclomatic complexity check(by gocyclo) because it will not affect readability.
func setupControllers(ctx context.Context,
	mgr controllerruntime.Manager,
	opts *options.Options,
) {
	controllerContext := controllerscontext.Context{
		Mgr: mgr,
		Opts: controllerscontext.Options{
			Controllers:                  opts.Controllers,
			MultiControllers:             opts.MultiControllers,
			ClusterStatusUpdateFrequency: opts.ClusterStatusUpdateFrequency,
			ClusterSuccessThreshold:      opts.ClusterSuccessThreshold,
			ClusterFailureThreshold:      opts.ClusterFailureThreshold,
			ClusterAPIQPS:                opts.ClusterAPIQPS,
			ClusterAPIBurst:              opts.ClusterAPIBurst,
			ConcurrentWorkSyncs:          opts.ConcurrentWorkSyncs,
			DebugMode:                    opts.DebugMode,
		},
		StopChan: ctx.Done(),
	}
	if err := controllers.StartControllers(controllerContext, controllersDisabledByDefault); err != nil {
		klog.ErrorS(err, "Failed to start controllers")
	}
}
