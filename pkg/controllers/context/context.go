package context

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Options defines all the parameters required by our controllers.
type Options struct {
	// Controllers contains all controller names.
	Controllers []string
	// MultiControllers is the list of member cluster controllers to enable or disable
	// '*' means "all enabled by default controllers"
	// 'foo' means "enable 'foo'"
	// '-foo' means "disable 'foo'"
	// first item for a particular name wins
	MultiControllers []string
	// ClusterStatusUpdateFrequency is the frequency that controller computes and report cluster status.
	// It must work with ClusterMonitorGracePeriod.
	ClusterStatusUpdateFrequency metav1.Duration
	// ClusterSuccessThreshold is the duration of successes for the cluster to be considered healthy after recovery.
	ClusterSuccessThreshold metav1.Duration
	// ClusterFailureThreshold is the duration of failure for the cluster to be considered unhealthy.
	ClusterFailureThreshold metav1.Duration
	// ClusterAPIQPS is the QPS to use while talking with cluster kube-apiserver.
	ClusterAPIQPS float32
	// ClusterAPIBurst is the burst to allow while talking with cluster kube-apiserver.
	ClusterAPIBurst int
	// ConcurrentWorkSyncs is the number of Works that are allowed to sync concurrently.
	ConcurrentWorkSyncs int
	// DebugMode indicates whether to enable debug mode.
	DebugMode bool
}

// Context defines the context object for controller.
type Context struct {
	Mgr              controllerruntime.Manager
	Opts             Options
	StopChan         <-chan struct{}
	DynamicClientSet dynamic.Interface
}

// IsControllerEnabled check if a specified controller enabled or not.
func (c Context) IsControllerEnabled(name string, disabledByDefaultControllers sets.Set[string]) bool {
	hasStar := false
	for _, ctrl := range c.Opts.Controllers {
		if ctrl == name {
			return true
		}
		if ctrl == "-"+name {
			return false
		}
		if ctrl == "*" {
			hasStar = true
		}
	}
	// if we get here, there was no explicit choice
	if !hasStar {
		// nothing on by default
		return false
	}

	return !disabledByDefaultControllers.Has(name)
}

// InitFunc is used to launch a particular controller.
// Any error returned will cause the controller process to `Fatal`
// The bool indicates whether the controller was enabled.
type InitFunc func(ctx Context) (enabled bool, err error)

// Initializers is a public map of named controller groups.
type Initializers map[string]InitFunc

// ControllerNames returns all known controller names.
func (i Initializers) ControllerNames() []string {
	return sets.StringKeySet(i).List()
}

// StartControllers starts a set of controllers with a specified ControllerContext.
func (i Initializers) StartControllers(ctx Context, controllersDisabledByDefault sets.Set[string]) error {
	for controllerName, initFn := range i {
		if !ctx.IsControllerEnabled(controllerName, controllersDisabledByDefault) {
			klog.V(2).InfoS("controller is disabled", "name", controllerName)
			continue
		}
		klog.V(3).InfoS("Starting controller", "name", controllerName)
		started, err := initFn(ctx)
		if err != nil {
			klog.ErrorS(err, "Error starting controller", "name", controllerName)
			return err
		}
		if !started {
			klog.V(2).InfoS("Skipping controller", "name", controllerName)
			continue
		}
		klog.V(3).InfoS("Started controller", "name", controllerName)
	}
	return nil
}
