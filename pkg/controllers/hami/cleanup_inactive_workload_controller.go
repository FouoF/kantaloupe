package hami

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/dynamia-ai/kantaloupe/pkg/service/monitoring"
)

const (
	CleanupInactiveWorkloadControllerName = "%s-cleanup-inactive-workload-controller"
	CleanupInactiveWorkloadPeriod         = time.Second * 5
)

type CleanupInactiveWorkloadController struct {
	Cluster string
	client.Client
	MonitoringService                monitoring.Service
	CleanupInactiveWorkloadThreshold time.Duration
	EventRecorder                    record.EventRecorder
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *CleanupInactiveWorkloadController) Reconcile(_ context.Context, _ controllerruntime.Request) (controllerruntime.Result, error) {
	return controllerruntime.Result{}, nil
}

func (c *CleanupInactiveWorkloadController) Start(ctx context.Context) error {
	klog.InfoS("Starting cleanup inactive workload controller")
	defer klog.InfoS("Shutting cleanup inactive workload controller")

	// Sync global cluster cr.
	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		vecs, err := c.MonitoringService.QueryVector(ctx, fmt.Sprintf(`Device_last_kernel_of_container{cluster="%s"}`, c.Cluster))
		if err != nil || len(vecs) == 0 {
			klog.ErrorS(err, "Failed to get metrics")
			return
		}

		for _, vec := range vecs {
			val := vec.Value
			threshold := c.CleanupInactiveWorkloadThreshold.Seconds()
			if float64(val) > threshold {
				podName := vec.Metric["podname"]
				namespace := vec.Metric["podnamespace"]

				pod := &corev1.Pod{}
				if err := c.Get(ctx, client.ObjectKey{Namespace: string(namespace), Name: string(podName)}, pod); err != nil {
					klog.ErrorS(err, "Failed to get pod", "podName", string(podName), "namespace", string(namespace))
					return
				}
				if err := c.cleanupWorkloadOwner(ctx, pod); err != nil {
					klog.ErrorS(err, "Failed to cleanup workload", "podName", string(podName), "namespace", string(namespace))
				}
			}
		}
	}, CleanupInactiveWorkloadPeriod)

	<-ctx.Done()
	return nil
}

func (c *CleanupInactiveWorkloadController) cleanupWorkloadOwner(ctx context.Context, pod *corev1.Pod) error {
	if len(pod.OwnerReferences) == 0 {
		if err := c.Delete(ctx, pod); err != nil {
			return err
		}
	}

	// TODO: only delete deployment.
	for _, owner := range pod.OwnerReferences {
		name := owner.Name
		namespace := pod.GetNamespace()
		if owner.Kind == "ReplicaSet" {
			rs := &appsv1.ReplicaSet{}
			if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, rs); err != nil {
				return err
			}
			for _, rsOwner := range rs.OwnerReferences {
				if rsOwner.Kind != "Deployment" {
					continue
				}

				deploy := &appsv1.Deployment{}
				if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: rsOwner.Name}, deploy); err != nil {
					return err
				}
				return c.Delete(ctx, deploy)
			}
		}
	}

	return nil
}

// SetupWithManager creates a controller and register to controller manager.
func (c *CleanupInactiveWorkloadController) SetupWithManager(mgr controllerruntime.Manager) error {
	return utilerrors.NewAggregate([]error{
		// controllerruntime.NewControllerManagedBy(mgr).
		// 	For(&corev1.Pod{}).
		// 	Named(fmt.Sprintf(CleanupInactiveWorkloadControllerName, c.Cluster)).
		// 	Complete(c),
		mgr.Add(c),
	})
}
