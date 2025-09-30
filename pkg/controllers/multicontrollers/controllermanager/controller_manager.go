package controllermanager

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	clustercrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
	kfv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/controllers/gateway"
	"github.com/dynamia-ai/kantaloupe/pkg/controllers/hami"
	"github.com/dynamia-ai/kantaloupe/pkg/controllers/kantaloupeflow"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/service/monitoring"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/env"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/gclient"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/helper"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/portallocate"
)

const (
	// ControllerName is the controller name that will be used when reporting events.
	ControllerName = "multi-cluster-controller-manager"

	LastAppliedConfigurationAnnotation = "kubectl.kubernetes.io/last-applied-configuration"

	MultiControllerFinalizer = "kantaloupe.dynamic.io/multi-controller"
)

type InitController map[string]MemberController

var RegisteredMultiControllers = make(InitController)

// ControllerNames returns all known controller names.
func (i InitController) ControllerNames() []string {
	return sets.StringKeySet(i).List()
}

type MemberControllerCleanup interface {
	Cleanup() error
}

func init() {
	RegisteredMultiControllers = InitController{
		"kantaloupeflowController":           startKantaloueflowController,
		"kantaloupeflowDeploymentController": startKantaloueflowDeploymentController,
		"restartDevicePluginController":      startRestartDevicePluginController,
		"cleanupInactiveWorkloadController":  startCleanupInactiveWorkloadController,
		"podGPUMemScaleController":           startPodGPUMemScaleController,
		"gatewaysectionControllerController": startGatewaysectionControllerController,
	}
}

type MemberController func(ctx context.Context, c *Controller, mgr controllerruntime.Manager, cluster *clustercrdv1alpha1.Cluster, allocate portallocate.Allocate) (interface{}, error)

// Controller is to sync Cluster.
type Controller struct {
	client.Client // used to operate Cluster resources.
	EventRecorder record.EventRecorder
	// ClusterKubernetesClientFunc func(ctx context.Context, runtimeClient client.Client, clusterName string) (kubeclient.Client, error)
	GlobalManger controllerruntime.Manager

	// ClusterClientOption holds the attributes that should be injected to a Kubernetes client.
	ClusterClientOption *utils.ClientOption

	ClusterKubeconfig func(clusterName string, client client.Client, clientOption *utils.ClientOption) (*rest.Config, error)

	// Queue is an listeningQueue that listens for events from Informers and adds object keys to
	// the Queue for processing
	Queue                   workqueue.RateLimitingInterface
	ControllerManager       sync.Map
	ControllersGenericEvent map[string]chan event.GenericEvent
	MultiControllers        []string
	// ConcurConcurrentWorkSyncsrentWorkSyncs is the number of MultiClusterSyncs that are allowed to sync concurrently.
	ConcurrentWorkSyncs int
}

type ControllerManager struct {
	cancelFunc     context.CancelFunc
	secretRef      clustercrdv1alpha1.LocalSecretReference
	cleanupFuncMap map[string]MemberControllerCleanup
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *Controller) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.V(4).InfoS("Reconciling cluster", "cluster", req.NamespacedName.Name)

	cluster := &clustercrdv1alpha1.Cluster{}
	if err := c.Get(ctx, req.NamespacedName, cluster); err != nil {
		// The resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			c.StopControllerManager(cluster.Name)
			return controllerruntime.Result{}, nil
		}
		return controllerruntime.Result{Requeue: true}, err
	}

	if !cluster.DeletionTimestamp.IsZero() {
		if err := c.CleanupBeforeStop(cluster.Name); err != nil {
			return controllerruntime.Result{}, err
		}
		c.StopControllerManager(cluster.Name)
		return controllerruntime.Result{}, c.removeFinalizer(ctx, cluster)
	}

	if !helper.IsClusterReady(&cluster.Status) {
		klog.ErrorS(nil, "cluster not ready, retry again", "cluster", cluster.Name)
		c.StopControllerManager(cluster.Name)
		return controllerruntime.Result{}, nil
	}

	manager, ok := c.ControllerManager.Load(cluster.Name)
	if ok {
		klog.V(4).InfoS("cluster is already added", "cluster", cluster.Name)
		m := manager.(*ControllerManager)
		secretRef := cluster.Spec.SecretRef
		if equality.Semantic.DeepEqual(m.secretRef, *secretRef) {
			klog.V(4).InfoS("Event was detected but nothing changed for cluster", "cluster", cluster.Name)
			return controllerruntime.Result{}, nil
		}
		klog.V(2).InfoS("Secret ref is changed,rebuild the object", "cluster", cluster.Name, "old", m.secretRef, "new", *secretRef)
		c.StopControllerManager(cluster.Name)
	}

	return c.StartControllerManager(ctx, cluster)
}

func (c *Controller) Start(ctx context.Context) error {
	go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	<-ctx.Done()
	return nil
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := c.Queue.Get()
	if shutdown {
		klog.V(2).InfoS("Queue already closed")
		return false
	}

	err := func(obj interface{}) error {
		req, ok := obj.(reconcile.Request)
		if !ok {
			c.Queue.Forget(obj)
			return nil
		}

		if _, err := c.Reconcile(ctx, req); err != nil {
			return fmt.Errorf("error syncing '%s': %s, requeuing", req.Name, err.Error())
		}
		c.Queue.Done(req)
		return nil
	}(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func removeLastAppliedConfigurationAnnotation(annotations map[string]string) {
	delete(annotations, LastAppliedConfigurationAnnotation)
}

// SetupWithManager creates a controller and register to controller manager.
func (c *Controller) SetupWithManager(mgr controllerruntime.Manager) error {
	c.GlobalManger = mgr
	return utilerrors.NewAggregate([]error{
		controllerruntime.NewControllerManagedBy(mgr).
			For(&clustercrdv1alpha1.Cluster{}).
			Named(ControllerName).
			WithOptions(controller.Options{
				MaxConcurrentReconciles: c.ConcurrentWorkSyncs,
			}).Complete(c),
		mgr.Add(c),
	})
}

// StripUnusedFields is the transform function for shared informers,
// it removes unused fields from objects before they are stored in the cache to save memory.
func StripUnusedFields(obj interface{}) (interface{}, error) {
	if tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}

	accessor, err := meta.Accessor(obj)
	if err != nil {
		// shouldn't happen
		//lint:ignore nilerr reason
		return obj, nil
	}
	// ManagedFields is large and we never use it
	accessor.SetManagedFields(nil)

	// kubectl.kubernetes.io/last-applied-configuration Fields is large and we never use it
	removeLastAppliedConfigurationAnnotation(accessor.GetAnnotations())
	return obj, nil
}

func (c *Controller) isControllerEnabled(name string) bool {
	hasStar := false
	for _, ctrl := range c.MultiControllers {
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
	return hasStar
}

func (c *Controller) StartControllerManager(ctx context.Context, cluster *clustercrdv1alpha1.Cluster) (controllerruntime.Result, error) {
	config, err := c.ClusterKubeconfig(cluster.GetName(), c.Client, c.ClusterClientOption)
	if err != nil {
		klog.ErrorS(err, "Get kube config form cluster", "cluster", cluster.Name)
		return controllerruntime.Result{RequeueAfter: 10 * time.Second}, nil
	}

	controllerManager, err := controllerruntime.NewManager(config, controllerruntime.Options{
		Scheme: gclient.NewSchema(),
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		Cache: cache.Options{
			// ref: https://github.com/projectcontour/contour/blob/main/cmd/contour/serve.go#L252
			// DefaultTransform is called for objects that do not have a TransformByObject function.
			DefaultTransform: StripUnusedFields,
		},
	})
	if err != nil {
		klog.ErrorS(err, "Failed to build  controller manager", "cluster", cluster.Name)
		return controllerruntime.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// install kantaloupe CRD
	if err := c.createCRDs(ctx, config, cluster); err != nil {
		return controllerruntime.Result{}, err
	}

	allocate, err := portallocate.New(ctx, gclient.NewForConfigOrDie(config))
	if err != nil {
		klog.ErrorS(err, "Failed to create port allocator")
		return controllerruntime.Result{}, err
	}

	cleanupFuncMap := make(map[string]MemberControllerCleanup)
	for controllerName, initFn := range RegisteredMultiControllers {
		started := c.isControllerEnabled(controllerName)
		if !started {
			klog.V(3).InfoS("Controller is disabled", "cluster", cluster.Name, "multi-controller", controllerName)
			continue
		}

		klog.V(2).InfoS("Starting multi-controller for cluster", "cluster", cluster.Name, "multi-controller", controllerName)
		memberController, err := initFn(ctx, c, controllerManager, cluster, allocate)
		if err != nil {
			klog.ErrorS(err, "Error starting", "multi-controller", controllerName, "cluster", cluster.Name)
			return controllerruntime.Result{RequeueAfter: 10 * time.Second}, nil
		}
		// if the controller implements MemberControllerCleanup to do some cleanup before controller-manager stopped, call the cleanup function
		if cleaner, ok := memberController.(MemberControllerCleanup); ok {
			cleanupFuncMap[controllerName] = cleaner
		}

		klog.V(2).InfoS("Started multi-controller for cluster", "cluster", cluster.Name, "multi-controller", controllerName)
	}

	_, ok := c.ControllerManager.Load(cluster.Name)
	if ok {
		return controllerruntime.Result{}, nil
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	c.ControllerManager.Store(cluster.Name, &ControllerManager{
		cancelFunc:     cancelFunc,
		secretRef:      *cluster.Spec.SecretRef,
		cleanupFuncMap: cleanupFuncMap,
	})

	go func() {
		// blocks until the context is done.
		if err := controllerManager.Start(ctx); err != nil {
			c.ControllerManager.Delete(cluster.Name)
			c.Queue.AddRateLimited(controllerruntime.Request{NamespacedName: types.NamespacedName{
				Name: cluster.Name,
			}})
			klog.ErrorS(err, "Controller manager exits unexpectedly", "cluster", cluster.Name)
			return
		}
	}()

	return controllerruntime.Result{}, c.addFinalizer(ctx, cluster)
}

func (c *Controller) createCRDs(ctx context.Context, config *rest.Config, cluster *clustercrdv1alpha1.Cluster) error {
	clusterClient, err := client.New(config, client.Options{
		Scheme: gclient.NewSchema(),
	})
	if err != nil {
		return err
	}

	// get kantaloupeflow crd from global cluster.
	name := fmt.Sprintf("%s.%s", "kantaloupeflows", kfv1alpha1.GroupName)
	globalCRD := &apiextensionsv1.CustomResourceDefinition{}
	if err := c.Get(ctx, client.ObjectKey{Name: name}, globalCRD); err != nil {
		klog.ErrorS(err, "failed to get global cluster crd", "cluster", "global-cluster", "crd", name)
		return err
	}

	// get kantaloupeflow crd from sub cluster.
	crd := &apiextensionsv1.CustomResourceDefinition{}
	err = clusterClient.Get(ctx, client.ObjectKey{Name: name}, crd)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			klog.ErrorS(err, "failed to get sub cluster crd kantaloupeflow", "cluster", klog.KObj(cluster))
			return err
		}
		globalCRD.ResourceVersion = ""
		globalCRD.Status = apiextensionsv1.CustomResourceDefinitionStatus{}
		if err := clusterClient.Create(ctx, globalCRD); err != nil {
			klog.ErrorS(err, "failed to create crd", "cluster", klog.KObj(cluster))
			return err
		}
		return nil
	}

	if !equality.Semantic.DeepEqual(globalCRD.Spec, crd.Spec) {
		crd.Spec = globalCRD.Spec
		if err := clusterClient.Update(ctx, crd); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) removeFinalizer(ctx context.Context, cluster *clustercrdv1alpha1.Cluster) error {
	finalizersUpdated := controllerutil.RemoveFinalizer(cluster, MultiControllerFinalizer)
	if finalizersUpdated {
		return c.Update(ctx, cluster)
	}
	return nil
}

func (c *Controller) addFinalizer(ctx context.Context, cluster *clustercrdv1alpha1.Cluster) error {
	finalizersUpdated := controllerutil.AddFinalizer(cluster, MultiControllerFinalizer)
	if finalizersUpdated {
		return c.Update(ctx, cluster)
	}
	return nil
}

func (c *Controller) StopControllerManager(name string) {
	manger, exist := c.GetControllerManager(name)
	if !exist {
		return
	}

	manger.cancelFunc()
	c.ControllerManager.Delete(name)
}

func (c *Controller) CleanupBeforeStop(name string) error {
	manger, exist := c.GetControllerManager(name)
	if !exist {
		klog.V(2).InfoS("Controller manager already stopped", "cluster", name)
		return nil
	}

	for controllerName, cleaner := range manger.cleanupFuncMap {
		klog.V(2).InfoS("Run multi-controller cleanup for cluster", "controllerName", controllerName, "cluster", name)
		if err := cleaner.Cleanup(); err != nil {
			klog.ErrorS(err, "Run multi-controller cleanup for cluster failed", "controllerName", controllerName, "cluster", name)
			return err
		}
	}
	return nil
}

func (c *Controller) GetControllerManager(name string) (*ControllerManager, bool) {
	value, ok := c.ControllerManager.Load(name)
	if !ok {
		return nil, false
	}

	manager, ok := value.(*ControllerManager)
	if !ok {
		return nil, false
	}
	return manager, true
}

func startKantaloueflowController(_ context.Context, ctr *Controller, mgr controllerruntime.Manager, cluster *clustercrdv1alpha1.Cluster, allocate portallocate.Allocate) (interface{}, error) {
	kantaloupeflowController := &kantaloupeflow.Controller{
		Cluster:            cluster.Name,
		LocalClusterClient: ctr.Client,
		Client:             mgr.GetClient(),
		PortAllocate:       allocate,
		EventRecorder:      mgr.GetEventRecorderFor(fmt.Sprintf(kantaloupeflow.ControllerName, cluster.Name)),
	}

	if err := kantaloupeflowController.SetupWithManager(mgr); err != nil {
		klog.ErrorS(err, "failed to setup kantaloupeflow controller", "cluster", cluster.Name)
	}

	return kantaloupeflowController, nil
}

func startKantaloueflowDeploymentController(_ context.Context, _ *Controller, mgr controllerruntime.Manager, cluster *clustercrdv1alpha1.Cluster, _ portallocate.Allocate) (interface{}, error) {
	deploymentController := &kantaloupeflow.DeplymentController{
		Cluster:       cluster.Name,
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor(fmt.Sprintf(kantaloupeflow.DeploymentControllerName, cluster.Name)),
	}

	if err := deploymentController.SetupWithManager(mgr); err != nil {
		klog.ErrorS(err, "failed to setup kantaloupeflow deployment controller", "cluster", cluster.Name)
	}

	return deploymentController, nil
}

func startRestartDevicePluginController(_ context.Context, _ *Controller, mgr controllerruntime.Manager, cluster *clustercrdv1alpha1.Cluster, _ portallocate.Allocate) (interface{}, error) {
	restartDevicePluginController := &hami.RestartDevicePluginController{
		Cluster:       cluster.Name,
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor(fmt.Sprintf(hami.RestartDevicePluginControllerName, cluster.Name)),
	}

	if err := restartDevicePluginController.SetupWithManager(mgr); err != nil {
		klog.ErrorS(err, "failed to setup restart device plugin controller", "cluster", cluster.Name)
	}

	return restartDevicePluginController, nil
}

func startPodGPUMemScaleController(_ context.Context, _ *Controller, mgr controllerruntime.Manager, cluster *clustercrdv1alpha1.Cluster, _ portallocate.Allocate) (interface{}, error) {
	podGPUMemScaleController := &hami.PodGPUMemScaleController{
		Cluster:       cluster.Name,
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor(fmt.Sprintf(hami.PodGPUMemScaleControllerName, cluster.Name)),
	}

	if err := podGPUMemScaleController.SetupWithManager(mgr); err != nil {
		klog.ErrorS(err, "failed to setup restart pod gpu mem scale controller", "cluster", cluster.Name)
	}

	return podGPUMemScaleController, nil
}

func startCleanupInactiveWorkloadController(_ context.Context, _ *Controller, mgr controllerruntime.Manager, cluster *clustercrdv1alpha1.Cluster, _ portallocate.Allocate) (interface{}, error) {
	monitoringEngine, err := engine.NewPrometheusClient(cluster.Spec.PrometheusAddress)
	if err != nil {
		return nil, err
	}
	service := monitoring.NewService(monitoringEngine)

	thresholdStr := env.CleanupInactiveWorkloadThreshold.Get()
	threshold, _ := strconv.Atoi(thresholdStr)

	cleanupInactiveWorloadController := &hami.CleanupInactiveWorkloadController{
		Cluster:                          cluster.Name,
		Client:                           mgr.GetClient(),
		MonitoringService:                service,
		CleanupInactiveWorkloadThreshold: time.Second * time.Duration(threshold),
		EventRecorder:                    mgr.GetEventRecorderFor(fmt.Sprintf(hami.CleanupInactiveWorkloadControllerName, cluster.Name)),
	}

	if err := cleanupInactiveWorloadController.SetupWithManager(mgr); err != nil {
		klog.ErrorS(err, "failed to setup cleanup inactive workload controller", "cluster", cluster.Name)
	}

	return cleanupInactiveWorloadController, nil
}

func startGatewaysectionControllerController(_ context.Context, _ *Controller, mgr controllerruntime.Manager, cluster *clustercrdv1alpha1.Cluster, allocate portallocate.Allocate) (interface{}, error) {
	cleanupInactiveWorloadController := &gateway.SectionController{
		Cluster:       cluster.Name,
		Client:        mgr.GetClient(),
		PortAllocate:  allocate,
		EventRecorder: mgr.GetEventRecorderFor(fmt.Sprintf(gateway.ControllerName, cluster.Name)),
	}

	if err := cleanupInactiveWorloadController.SetupWithManager(mgr); err != nil {
		klog.ErrorS(err, "failed to setup gateway section controller", "cluster", cluster.Name)
	}

	return cleanupInactiveWorloadController, nil
}
