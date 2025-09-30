package monitoring

import (
	"fmt"
	"strings"

	prometheusmodel "github.com/prometheus/common/model"
)

func buildClusterQuery(cluster string, queryType QueryType) string {
	labels := map[string]string{
		"cluster": cluster,
	}
	return fmt.Sprintf(`kantaloupe_cluster_%s%s`, queryType, buildLabelMatchers(labels))
}

func buildNodeQuery(cluster, node string, queryType QueryType) string {
	labels := map[string]string{
		"cluster": cluster,
		"node":    node,
	}
	return fmt.Sprintf(`kantaloupe_node_%s%s`, queryType, buildLabelMatchers(labels))
}

func buildWorkLoadQuery(cluster, node, namespace, name string, queryType QueryType) string {
	labels := map[string]string{
		"cluster": cluster,
		"node":    node,
	}

	switch queryType {
	case QueryTypeMemoryUsed, QueryTypeMemoryAllocated, QueryTypeCPUUsed, QueryTypeCPUAllocated:
		if namespace != "" {
			labels["namespace"] = namespace
		}
		if name != "" {
			labels["pod"] = fmt.Sprintf(`~"%s.+"`, name)
		}
	default:
		if name != "" && namespace != "" {
			labels["deployment"] = fmt.Sprintf("%s/%s", namespace, name)
		}
	}

	return fmt.Sprintf("kantaloupe_workload_%s%s", queryType, buildLabelMatchers(labels))
}

func buildGPUQuery(cluster, node, vendor, model, uuid string, queryType GPUQueryType) string {
	labels := map[string]string{
		"cluster":   cluster,
		"node":      node,
		"vendor":    vendor,
		"modelName": model,
		"UUID":      uuid,
	}
	if queryType == GPUQueryTypeCount {
		return fmt.Sprintf(`count(kantaloupe_gpu_temp%s)`, buildLabelMatchers(labels))
	}
	return fmt.Sprintf(`kantaloupe_gpu_%s%s`, queryType, buildLabelMatchers(labels))
}

func buildPlatformQuery(queryType QueryType) string {
	return fmt.Sprintf(`kantaloupe_global_%s`, queryType)
}

func buildLabelMatchers(labels map[string]string) string {
	matchers := make([]string, 0, len(labels))
	for k, v := range labels {
		if v == "" {
			continue
		}
		// The prefix of regenx should not surrounded with "".
		if v[0] == '~' {
			matchers = append(matchers, fmt.Sprintf(`%s=%s`, k, v))
		} else {
			matchers = append(matchers, fmt.Sprintf(`%s="%s"`, k, v))
		}
	}
	if len(matchers) == 0 {
		return ""
	}
	return "{" + strings.Join(matchers, ", ") + "}"
}

// If we can assert the result to be a single element, we can safely using index 0.
func assertOneElement(vec prometheusmodel.Vector, query string) error {
	if len(vec) != 1 {
		return fmt.Errorf("the query %s expects one element, got %d; please check the status of metrics service", query, len(vec))
	}
	return nil
}

func surroundWithSum(s string) string {
	return "sum(" + s + ")"
}
