package cluster

import (
	"context"
	"fmt"
	"strings"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	clustercrdv1alpha1 "github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1"
)

func (c *Controller) RunPrometheusConfigLoop(ctx context.Context) {
	if err := c.ReconcileAllScrapeConfigs(ctx); err != nil {
		klog.ErrorS(err, "error reconciling all scrape configs")
	}
}

func (c *Controller) ReconcileAllScrapeConfigs(ctx context.Context) error {
	var clusters clustercrdv1alpha1.ClusterList
	if err := c.Client.List(ctx, &clusters); err != nil {
		return err
	}

	for _, cluster := range clusters.Items {
		if cluster.Name == "local-cluster" {
			continue
		}
		if err := c.CreateOrUpdateScrapeConfigForCluster(ctx, &cluster); err != nil {
			klog.ErrorS(err, "failed to create/update scrapeconfig for cluster", "cluster", cluster.Name)
		}
	}

	return nil
}

func (c *Controller) CreateOrUpdateScrapeConfigForCluster(ctx context.Context, cluster *clustercrdv1alpha1.Cluster) error {
	name := "kantaloupe-federate-" + cluster.Name
	namespace := "monitoring"

	var existing monitoringv1alpha1.ScrapeConfig
	err := c.Client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &existing)
	if err == nil {
		existing.Spec.StaticConfigs = []monitoringv1alpha1.StaticConfig{
			{Targets: []monitoringv1alpha1.Target{monitoringv1alpha1.Target(RemoveProtocolPrefix(cluster.Spec.PrometheusAddress))}},
		}
		existing.Spec.RelabelConfigs = []monitoringv1.RelabelConfig{
			{TargetLabel: "cluster", Replacement: &cluster.Name},
		}
		return c.Client.Update(ctx, &existing)
	}

	if !apierrors.IsNotFound(err) {
		return err
	}

	template := &monitoringv1alpha1.ScrapeConfig{}
	if err := c.Client.Get(ctx, types.NamespacedName{
		Name:      "kantaloupe-local-cluster",
		Namespace: namespace,
	}, template); err != nil {
		return err
	}

	newCfg := template.DeepCopy()
	newCfg.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    map[string]string{"release": "prometheus"},
		OwnerReferences: []metav1.OwnerReference{
			*metav1.NewControllerRef(cluster, schema.GroupVersionKind{
				Group:   clustercrdv1alpha1.GroupVersion.Group,
				Version: clustercrdv1alpha1.GroupVersion.Version,
				Kind:    "Cluster",
			}),
		},
	}
	newCfg.Spec.StaticConfigs = []monitoringv1alpha1.StaticConfig{
		{Targets: []monitoringv1alpha1.Target{monitoringv1alpha1.Target(RemoveProtocolPrefix(cluster.Spec.PrometheusAddress))}},
	}
	newCfg.Spec.RelabelConfigs = []monitoringv1.RelabelConfig{
		{TargetLabel: "cluster", Replacement: &cluster.Name},
	}

	return c.Client.Create(ctx, newCfg)
}

func (c *Controller) DeleteScrapeConfig(ctx context.Context, cluster *clustercrdv1alpha1.Cluster) error {
	name := fmt.Sprintf("kantaloupe-federate-%s", cluster.Name)
	namespace := "monitoring"
	if err := c.Client.Delete(ctx, &monitoringv1alpha1.ScrapeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		klog.ErrorS(err, "failed to delete scrape config", "cluster", klog.KObj(cluster))
		return err
	}
	return nil
}

func RemoveProtocolPrefix(address string) string {
	if after, ok := strings.CutPrefix(address, "https://"); ok {
		return after
	}
	if after, ok := strings.CutPrefix(address, "http://"); ok {
		return after
	}
	return address
}
