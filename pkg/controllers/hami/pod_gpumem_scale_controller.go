package hami

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kfv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/annotations"
)

const (
	PodGPUMemScaleControllerName = "%s-pod-gpumem-scale-controller"

	PodAllocationAnnotation = "kantaloupe.dynamia.ai/pod-allocation-meet"

	OOMExpansionAnnotation = "kantaloupe.dynamia.ai/oom-expansion-to"

	KantaloupeGpusAnnotation = "kantaloupe.dynamia.ai/gpus"
)

type PodGPUMemScaleController struct {
	Cluster string
	client.Client
	EventRecorder record.EventRecorder
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *PodGPUMemScaleController) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.V(4).InfoS("Reconciling Kantaloupeflowpod", "KObj", req.NamespacedName.String())

	pod := &corev1.Pod{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, pod); err != nil {
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		return controllerruntime.Result{}, err
	}

	flow := &kfv1alpha1.KantaloupeFlow{}
	if err := c.Client.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: pod.Labels[constants.KantaloupeFlowAppLabelKey]}, flow); err != nil {
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		return controllerruntime.Result{}, err
	}

	if pod.Annotations == nil {
		return controllerruntime.Result{}, nil
	}

	if _, ok := pod.Annotations[annotations.PodGPUMemoryAnnotation]; !ok {
		return controllerruntime.Result{}, nil
	}

	// example of hami.io/vgpu-devices-allocated: GPU-801f670d-758f-4d91-1b4c-abf475afe38f,NVIDIA,1024,10:;
	annotation := pod.Annotations[annotations.PodGPUMemoryAnnotation]
	if flow.Annotations == nil {
		flow.Annotations = map[string]string{}
	}
	flow.Annotations[PodAllocationAnnotation] = annotation
	allocations, err := annotations.MarshalGPUAllocationAnnotation(annotation)
	if err != nil {
		return controllerruntime.Result{}, err
	}

	if _, ok := pod.Annotations[annotations.OOMExpansionAnnotation]; ok {
		// OOM expanded
		flow.Annotations[OOMExpansionAnnotation] = strconv.FormatInt(allocations[0].Memory, 10)
		return controllerruntime.Result{}, c.Update(ctx, flow)
	}
	// Clear OOM flag
	flow.Annotations[OOMExpansionAnnotation] = ""
	if err = c.Update(ctx, flow); err != nil {
		return controllerruntime.Result{}, err
	}
	// Restarted Pod or updated by kantaloupeflow
	flowvalues := strings.Split(flow.Annotations[PodAllocationAnnotation], ",")
	if len(flowvalues) != 2 {
		klog.V(4).InfoS("Invalid annotation format", "annotation", PodAllocationAnnotation)
		return controllerruntime.Result{}, nil
	}
	// Updated by kantaloupeflow
	if flowvalues[0] == strconv.FormatInt(allocations[0].Memory, 10) {
		return controllerruntime.Result{}, nil
	}
	// Restarted Pod
	needMemory, err := strconv.ParseInt(flowvalues[0], 10, 64)
	if err != nil {
		klog.V(4).ErrorS(err, "Invalid annotation format", "annotation", PodAllocationAnnotation)
		return controllerruntime.Result{}, nil
	}
	for i := range allocations {
		allocations[i].Memory = needMemory
	}
	pod.Annotations[annotations.PodGPUMemoryAnnotation] = annotations.UnmarshalGPUAllocationAnnotation(allocations)
	return controllerruntime.Result{}, c.Update(ctx, pod)
}

// SetupWithManager creates a controller and register to controller manager.
func (c *PodGPUMemScaleController) SetupWithManager(mgr controllerruntime.Manager) error {
	configMapPredicateFunc := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			newer := createEvent.Object.(*corev1.Pod)
			if _, ok := newer.Labels[constants.KantaloupeFlowAppLabelKey]; ok {
				return true
			}
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			newer := updateEvent.ObjectNew.(*corev1.Pod)
			if _, ok := newer.Labels[constants.KantaloupeFlowAppLabelKey]; ok {
				return true
			}
			return false
		},
		DeleteFunc: func(event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}

	return controllerruntime.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}, builder.WithPredicates(configMapPredicateFunc)).
		Named(fmt.Sprintf(PodGPUMemScaleControllerName, c.Cluster)).
		Complete(c)
}
