package kantaloupeflow

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kfv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1"
	"github.com/dynamia-ai/kantaloupe/pkg/utils"
)

const (
	DeploymentControllerName = "%s-kantaloupeflow-deployment-controller"
)

type DeplymentController struct {
	Cluster string
	client.Client
	EventRecorder record.EventRecorder
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *DeplymentController) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.V(4).InfoS("Reconciling Deployment", "KObj", req.NamespacedName.String())
	deploy := &appsv1.Deployment{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, deploy); err != nil {
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		return controllerruntime.Result{}, err
	}

	return controllerruntime.Result{}, c.syncKantaloupeFlowStatus(ctx, deploy)
}

func (c *DeplymentController) syncKantaloupeFlowStatus(ctx context.Context, deploy *appsv1.Deployment) error {
	owners := deploy.GetOwnerReferences()

	objectKey := client.ObjectKey{}
	for _, owner := range owners {
		if owner.Kind == kfv1alpha1.KantaloupeFlowResourceKind {
			objectKey.Name = owner.Name
			objectKey.Namespace = deploy.GetNamespace()
		}
	}

	flow := &kfv1alpha1.KantaloupeFlow{}
	if err := c.Get(ctx, objectKey, flow); err != nil {
		klog.ErrorS(err, "the owner reference of deploy kantaloupeFlow is not found", "deployment", deploy)
		return nil
	}

	now := flow.DeepCopy()

	now.Status.Replicas = deploy.Status.Replicas
	now.Status.ReadyReplicas = deploy.Status.ReadyReplicas
	now.Status.Conditions = convertDeploymentCondition(deploy.Status.Conditions)

	if !equality.Semantic.DeepEqual(flow.Status, now.Status) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err := utils.UpdateStatus(ctx, c.Client, flow,
				func() error {
					flow.Status.Conditions = now.Status.Conditions
					flow.Status.Networking = now.Spec.Networking
					flow.Status.Replicas = now.Status.Replicas
					flow.Status.ReadyReplicas = now.Status.ReadyReplicas

					return nil
				})
			return err
		})
		if err != nil {
			klog.ErrorS(err, "Failed to update kantaloupeflow status", "kantaloupeflow", klog.KObj(flow))
			return err
		}
	}

	return nil
}

// SetupWithManager creates a controller and register to controller manager.
func (c *DeplymentController) SetupWithManager(mgr controllerruntime.Manager) error {
	clusterPredicateFunc := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			obj := createEvent.Object.(*appsv1.Deployment)
			return hasOwnerRef(obj)
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			obj := updateEvent.ObjectNew.(*appsv1.Deployment)
			return hasOwnerRef(obj)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			obj := deleteEvent.Object.(*appsv1.Deployment)
			return hasOwnerRef(obj)
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}

	return controllerruntime.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}, builder.WithPredicates(clusterPredicateFunc)).
		Named(fmt.Sprintf(DeploymentControllerName, c.Cluster)).
		Complete(c)
}

func hasOwnerRef(deploy *appsv1.Deployment) bool {
	for _, owner := range deploy.OwnerReferences {
		if owner.Kind == "KantaloupeFlow" {
			return true
		}
	}
	return false
}

func convertDeploymentCondition(conditions []appsv1.DeploymentCondition) []metav1.Condition {
	res := []metav1.Condition{}
	for _, cond := range conditions {
		res = append(res, metav1.Condition{
			Type:               string(cond.Type),
			Status:             metav1.ConditionStatus(cond.Status),
			Reason:             cond.Reason,
			Message:            cond.Message,
			LastTransitionTime: cond.LastTransitionTime,
		})
	}

	return res
}
