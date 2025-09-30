package engine

import (
	"context"
	"time"

	prometheusapi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prometheusmodel "github.com/prometheus/common/model"
	"k8s.io/klog/v2"

	"github.com/dynamia-ai/kantaloupe/pkg/utils/errs"
)

type PrometheusInterface interface {
	Query(ctx context.Context, query string, ts time.Time) (prometheusmodel.Value, error)
	QueryRange(ctx context.Context, query string, r prometheusv1.Range) (prometheusmodel.Value, error)
}

func newPrometheus(addr string) (prometheusapi.Client, error) {
	client, err := prometheusapi.NewClient(prometheusapi.Config{
		Address: addr,
	})
	if err != nil {
		klog.ErrorS(err, "failed to create prometheus client", "Error:", err)
		return nil, err
	}
	return client, nil
}

type PrometheusOptions struct {
	TimeOut time.Duration
}

func WithTimeOut(timeout time.Duration) func(*PrometheusOptions) {
	return func(o *PrometheusOptions) {
		o.TimeOut = timeout
	}
}

var _ PrometheusInterface = (*Prometheus)(nil)

type Prometheus struct {
	client prometheusapi.Client
	ops    *PrometheusOptions
}

func NewPrometheusClient(addr string, opts ...func(*PrometheusOptions)) (PrometheusInterface, error) {
	client, err := newPrometheus(addr)
	// FIXME: error handling
	// if err != nil {
	// 	return nil, err
	// }
	options := &PrometheusOptions{
		TimeOut: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(options)
	}

	return &Prometheus{client: client, ops: options}, err
}

func (p *Prometheus) Query(ctx context.Context, query string, ts time.Time) (prometheusmodel.Value, error) {
	if p.client == nil {
		return nil, errs.ErrPrometheusClientUninitialized
	}
	ctx, cancel := context.WithTimeout(ctx, p.ops.TimeOut)
	defer cancel()
	api := prometheusv1.NewAPI(p.client)
	val, _, err := api.Query(ctx, query, ts)
	if err != nil {
		klog.ErrorS(err, "error query prometheus", "Error:", err)
		return nil, err
	}
	return val, nil
}

func (p *Prometheus) QueryRange(ctx context.Context, query string, r prometheusv1.Range) (prometheusmodel.Value, error) {
	if p.client == nil {
		return nil, errs.ErrPrometheusClientUninitialized
	}
	ctx, cancel := context.WithTimeout(ctx, p.ops.TimeOut)
	defer cancel()
	api := prometheusv1.NewAPI(p.client)
	val, _, err := api.QueryRange(ctx, query, r)
	if err != nil {
		klog.ErrorS(err, "error query range prometheus", "Error:", err)
		return nil, err
	}
	return val, nil
}
