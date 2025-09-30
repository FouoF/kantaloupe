package gateway

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/portallocate"
)

const (
	// ControllerName is the controller name that will be used when reporting events.
	ControllerName = "%s-gatewaysection-controller"

	// RequeueAfter is the time after which the controller will requeue the request.
	RequeueAfter = 3 * time.Second

	// SectionNameNotFount is the value to indicate the section name is not found in Gateway.
	SectionNameNotFount = -1

	// KantaloupeUTCPRouteFinalizer TCPRoute Finalizer.
	KantaloupeUTCPRouteFinalizer = "kantaloupe.io/tcproute-controller"
)

// SectionController is to sync Gateway SectionName.
type SectionController struct {
	Cluster       string
	client.Client // used to operate TCPRoute resources.
	EventRecorder record.EventRecorder
	PortAllocate  portallocate.Allocate
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *SectionController) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.V(4).InfoS("Reconciling TCPRoute Section", "KObj", req.NamespacedName.String())

	tcpRoute := &gatewayv1alpha2.TCPRoute{}
	if err := c.Client.Get(ctx, req.NamespacedName, tcpRoute); err != nil {
		// The resource no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		return controllerruntime.Result{}, err
	}

	if !tcpRoute.DeletionTimestamp.IsZero() {
		if err := c.removeGatewayListener(ctx, tcpRoute); err != nil {
			klog.ErrorS(err, "Failed to remove gateway listener", "KObj", klog.KObj(tcpRoute))
			return controllerruntime.Result{}, err
		}
		if err := c.removeFinalizer(ctx, tcpRoute); err != nil {
			klog.ErrorS(err, "failed to delete finalizer for tcpRoute", "tcpRoute", klog.KObj(tcpRoute))
			return controllerruntime.Result{}, err
		}

		return controllerruntime.Result{}, nil
	}
	if err := c.ensureFinalizer(ctx, tcpRoute); err != nil {
		klog.ErrorS(err, "faild to ensure finalizer for tcpRoute", "tcpRoute", klog.KObj(tcpRoute))
		return controllerruntime.Result{}, err
	}

	if err := c.syncGatewaySectionName(ctx, tcpRoute); err != nil {
		klog.ErrorS(err, "Failed to sync Gateway Section", "KObj", klog.KObj(tcpRoute))
		return controllerruntime.Result{}, err
	}
	return controllerruntime.Result{}, nil
}

func (c *SectionController) ensureFinalizer(ctx context.Context, tcproute *gatewayv1alpha2.TCPRoute) error {
	if ctrlutil.AddFinalizer(tcproute, KantaloupeUTCPRouteFinalizer) {
		return c.Client.Update(ctx, tcproute)
	}
	return nil
}

func (c *SectionController) removeFinalizer(ctx context.Context, tcproute *gatewayv1alpha2.TCPRoute) error {
	if ctrlutil.RemoveFinalizer(tcproute, KantaloupeUTCPRouteFinalizer) {
		return c.Client.Update(ctx, tcproute)
	}
	return nil
}

func (c *SectionController) syncGatewaySectionName(ctx context.Context, tcpRoute *gatewayv1alpha2.TCPRoute) error {
	for _, parentRef := range tcpRoute.Spec.ParentRefs {
		if *parentRef.Kind != gatewayv1alpha2.Kind("Gateway") {
			continue
		}

		gateway := &gatewayv1.Gateway{}
		namespace := tcpRoute.Namespace

		if parentRef.Namespace != nil {
			namespace = string(*parentRef.Namespace)
		}

		err := c.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: string(parentRef.Name)}, gateway)
		if err != nil {
			klog.ErrorS(err, "Failed to get Gateway", "Name", string(parentRef.Name), "Namespace", namespace, "TCPRoute", klog.KObj(tcpRoute))
			return err
		}
		klog.V(4).InfoS("Reconciling TCPRoute", "KObj", klog.KObj(tcpRoute), "Gateway", klog.KObj(gateway), "SectionName", *parentRef.SectionName)

		older := gateway.DeepCopy()
		buildGatewayListener(gateway, parentRef)

		// update gateway spec.
		if !equality.Semantic.DeepEqual(older.Spec, gateway.Spec) {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				newer := &gatewayv1.Gateway{}
				if err := c.Get(ctx, client.ObjectKeyFromObject(gateway), newer); err != nil {
					return err
				}
				gateway.ResourceVersion = newer.ResourceVersion

				return c.Update(ctx, gateway)
			})
			if err != nil {
				klog.ErrorS(err, "Failed to update gateway", "gateway", klog.KObj(gateway))
				return err
			}
		}
	}

	return nil
}

func (c *SectionController) removeGatewayListener(ctx context.Context, tcpRoute *gatewayv1alpha2.TCPRoute) error {
	if len(tcpRoute.Spec.ParentRefs) == 0 {
		klog.ErrorS(nil, "No parent refs found", "TCPRoute", klog.KObj(tcpRoute))
		return nil
	}

	gateway := &gatewayv1.Gateway{}
	parentRef := tcpRoute.Spec.ParentRefs[0]
	namespace := string(*parentRef.Namespace)
	name := string(parentRef.Name)
	err := c.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gateway)
	if err != nil {
		return err
	}

	idx := findSectionIndex(gateway, parentRef)
	if idx == SectionNameNotFount {
		return nil
	}
	gateway.Spec.Listeners = append(gateway.Spec.Listeners[:idx], gateway.Spec.Listeners[idx+1:]...)
	if err := c.Update(ctx, gateway); err != nil {
		klog.ErrorS(err, "Failed to update Gateway", "Gateway", klog.KObj(gateway))
		return err
	}

	// release port.
	_, port, _ := portallocate.SplitGatewaySectionName(string(*parentRef.SectionName))
	c.PortAllocate.Release(port)

	return nil
}

func buildGatewayListener(gateway *gatewayv1.Gateway, parentRef gatewayv1.ParentReference) {
	_, port, _ := portallocate.SplitGatewaySectionName(string(*parentRef.SectionName))

	var isExistListener bool
	for _, listener := range gateway.Spec.Listeners {
		if listener.Name == *parentRef.SectionName {
			isExistListener = true
			break
		}
	}

	if !isExistListener {
		gateway.Spec.Listeners = append(gateway.Spec.Listeners, gatewayv1.Listener{
			Name:     *parentRef.SectionName,
			Port:     gatewayv1.PortNumber(port),
			Protocol: gatewayv1.TCPProtocolType,
			AllowedRoutes: &gatewayv1.AllowedRoutes{
				Kinds: []gatewayv1.RouteGroupKind{
					{
						Kind: gatewayv1.Kind("TCPRoute"),
					},
				},
				Namespaces: &gatewayv1.RouteNamespaces{
					From: ptr.To(gatewayv1.NamespacesFromAll),
				},
			},
		})
	}
}

func findSectionIndex(gateway *gatewayv1.Gateway, parentRef gatewayv1.ParentReference) int {
	for idx, listener := range gateway.Spec.Listeners {
		if listener.Name == *parentRef.SectionName {
			return idx
		}
	}
	return SectionNameNotFount
}

// SetupWithManager creates a controller and register to controller manager.
func (c *SectionController) SetupWithManager(mgr controllerruntime.Manager) error {
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&gatewayv1alpha2.TCPRoute{}, builder.WithPredicates(LabelSelectorPredicate())).
		Named(ControllerName).
		Named(fmt.Sprintf(ControllerName, c.Cluster)).
		Complete(c)
}

func LabelSelectorPredicate() predicate.Predicate {
	ownerExist, _ := labels.NewRequirement(constants.KantaloupeFlowAppLabelKey, selection.Exists, nil)

	return predicate.NewPredicateFuncs(func(object client.Object) bool {
		return ownerExist.Matches(labels.Set(object.GetLabels()))
	})
}
