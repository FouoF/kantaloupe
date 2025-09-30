package portallocate

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/registry/core/service/allocator"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/dynamia-ai/kantaloupe/pkg/constants"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/env"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/helper"
)

var (
	ErrFull              = errors.New("range is full")
	ErrAllocated         = errors.New("provided port is already allocated")
	ErrMismatchedNetwork = errors.New("the provided port range does not match the current port range")
	ErrAvailable         = errors.New("no available port")
	ErrPortRange         = errors.New("port is not in the range")

	// ErrGatewaySectionNameInvalid is returned when the gateway section name is invalid.
	ErrGatewaySectionNameInvalid = errors.New("gateway section name is invalid")
)

type NotInRangeError struct {
	ValidPorts string
}

func (e *NotInRangeError) Error() string {
	return fmt.Sprintf("provided port is not in the valid range. The range of valid ports is %s", e.ValidPorts)
}

type Allocate interface {
	AllocateNext() (int, error)
	Release(port int) error
	Has(port int) bool
	Start(ctx context.Context) error
}

var _ Allocate = &PortAllocator{}

type PortAllocator struct {
	client.Client
	portRange *net.PortRange
	alloc     allocator.Interface
}

// Allocate reserves the port.
func (a *PortAllocator) Allocate(port int) error {
	ok, offset := a.contains(port)
	if !ok {
		// include valid port range in error
		validPorts := a.portRange.String()
		return &NotInRangeError{validPorts}
	}

	allocated, err := a.alloc.Allocate(offset)
	if err != nil {
		return err
	}
	if !allocated {
		return ErrAllocated
	}
	return nil
}

// AllocateNext returns an available port.
func (a *PortAllocator) AllocateNext() (int, error) {
	offset, ok, err := a.alloc.AllocateNext()
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrAvailable
	}
	return a.portRange.Base + offset, nil
}

// Release releases the port.
func (a *PortAllocator) Release(port int) error {
	ok, offset := a.contains(port)
	if !ok {
		return fmt.Errorf("%w: %d", ErrPortRange, port)
	}
	return a.alloc.Release(offset)
}

// Has returns true if the port is allocated.
func (a *PortAllocator) Has(port int) bool {
	ok, offset := a.contains(port)
	if !ok {
		return false
	}

	return a.alloc.Has(offset)
}

func (a *PortAllocator) ForEach(fn func(int)) {
	a.alloc.ForEach(func(offset int) {
		fn(a.portRange.Base + offset)
	})
}

func (a *PortAllocator) Size() int {
	return a.portRange.Size - a.alloc.Free()
}

func (a *PortAllocator) Start(ctx context.Context) error {
	klog.V(2).InfoS("Starting port allocator")

	ownerExist, err := labels.NewRequirement(constants.KantaloupeFlowAppLabelKey, selection.Exists, nil)
	if err != nil {
		return err
	}
	labelsSelector := labels.NewSelector().Add(*ownerExist)

	tclroutes := &gatewayv1alpha2.TCPRouteList{}
	if err = a.List(ctx, tclroutes, client.MatchingLabelsSelector{Selector: labelsSelector}); err != nil {
		return err
	}

	for _, tcproute := range tclroutes.Items {
		for _, ref := range tcproute.Spec.CommonRouteSpec.ParentRefs {
			if err = a.Allocate(int(*ref.Port)); err != nil {
				return err
			}
		}
	}

	gateway := &gatewayv1.Gateway{}
	if err := a.Get(ctx, client.ObjectKey{Name: "kantaloupe", Namespace: helper.GetCurrentNSOrDefault()}, gateway); err != nil {
		return err
	}

	for _, listener := range gateway.Spec.Listeners {
		_ = a.Allocate(int(listener.Port))
	}

	// Debug
	// go func() {
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		case <-time.After(30 * time.Second):
	// 			a.ForEach(func(port int) {
	// 				klog.V(4).InfoS("Allocated port", "port", port)
	// 			})
	// 			klog.V(4).InfoS("Allocated port detail",
	// 				"total", a.portRange.Size, "used", a.Size(), "free", a.alloc.Free())
	// 		}
	// 	}
	// }()

	return nil
}

// contains returns true and the offset if the port is in the range, and false
// and nil otherwise.
func (a *PortAllocator) contains(port int) (bool, int) {
	if !a.portRange.Contains(port) {
		return false, 0
	}

	offset := port - a.portRange.Base
	return true, offset
}

// New returns a new Allocate instance.
func New(_ context.Context, c client.Client) (Allocate, error) {
	portStart := env.GatewayEnvNamePortStart.Get()
	portCount := env.GatewayEnvNamePortCount.Get()

	netPortRange := net.ParsePortRangeOrDie(fmt.Sprintf("%d-%d", portStart, portStart+portCount))
	allocator := &PortAllocator{
		Client:    c,
		portRange: netPortRange,
		alloc:     allocator.NewAllocationMap(netPortRange.Size, netPortRange.String()),
	}

	allocator.ForEach(func(port int) {
		klog.V(4).InfoS("Allocated port", "port", port)
	})
	return allocator, nil
}

func SplitGatewaySectionName(sectionName string) (string, int, error) {
	section := strings.Split(sectionName, "--")
	if len(section) != 2 {
		return "", 0, ErrGatewaySectionNameInvalid
	}

	port, err := strconv.Atoi(section[1])
	return section[0], port, err
}
