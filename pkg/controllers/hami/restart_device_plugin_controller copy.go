package hami

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	RestartDevicePluginControllerName = "%s-restart-device-plugin-controller"
)

type RestartDevicePluginController struct {
	Cluster string
	client.Client
	EventRecorder record.EventRecorder
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *RestartDevicePluginController) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.V(4).InfoS("Reconciling ConfigMap", "KObj", req.NamespacedName.String())

	configMap := &corev1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, configMap); err != nil {
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		return controllerruntime.Result{}, err
	}

	return controllerruntime.Result{}, c.RestartDevicePlugin(ctx, configMap)
}

func (c *RestartDevicePluginController) RestartDevicePlugin(ctx context.Context, configMap *corev1.ConfigMap) error {
	// 1. get deamonset of hami device plugin.
	deamonsets := &appsv1.DaemonSetList{}
	err := c.List(ctx, deamonsets, &client.ListOptions{
		LabelSelector: labels.Set{"app.kubernetes.io/component": "hami-device-plugin"}.AsSelector(),
		Namespace:     configMap.GetNamespace(),
	})
	if err != nil {
		klog.ErrorS(err, "Failed to get hami device plugin")
		return err
	}

	if len(deamonsets.Items) == 0 {
		klog.InfoS("The hami device plugin is not found")
		return nil
	}

	for _, devicePlugin := range deamonsets.Items {
		dsCopy := devicePlugin.DeepCopy()

		// 2. check whether the device plugin is restarting. skip the action if
		// device plugin is restarting.
		if dsCopy.Status.NumberReady == 0 {
			klog.InfoS("the device plugin is restarting")
			return nil
		}

		// 3. restart the device plugin.
		if dsCopy.Spec.Template.Annotations == nil {
			dsCopy.Spec.Template.Annotations = map[string]string{}
		}
		dsCopy.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
		if err := c.Update(ctx, dsCopy); err != nil {
			klog.ErrorS(err, "failed to restart hami device plugin")
			return err
		}
	}

	return nil
}

// SetupWithManager creates a controller and register to controller manager.
func (c *RestartDevicePluginController) SetupWithManager(mgr controllerruntime.Manager) error {
	configMapPredicateFunc := predicate.Funcs{
		CreateFunc: func(event.CreateEvent) bool {
			// obj := createEvent.Object.(*corev1.ConfigMap)
			// _, ok := obj.Labels["app.kubernetes.io/component"]
			// return ok // TODO:
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			newer := updateEvent.ObjectNew.(*corev1.ConfigMap)
			val := newer.Labels["app.kubernetes.io/component"]
			if val == "hami-device-plugin" || val == "hami-scheduler" {
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
		For(&corev1.ConfigMap{}, builder.WithPredicates(configMapPredicateFunc)).
		Named(fmt.Sprintf(RestartDevicePluginControllerName, c.Cluster)).
		Complete(c)
}
