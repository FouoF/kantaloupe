package monitoring

import (
	"fmt"
	"sort"
	"time"

	prometheusmodel "github.com/prometheus/common/model"
)

type PrometheusVectorMap map[string]*prometheusmodel.Sample

// VectorToMapByLabel converts a Prometheus Vector to a map where the key is the value of a specific label and the value is the corresponding sample.
func VectorToMapByLabel(vec prometheusmodel.Vector, labelName string) (map[string]*prometheusmodel.Sample, error) {
	result := make(map[string]*prometheusmodel.Sample)

	for _, sample := range vec {
		labelValue := string(sample.Metric[prometheusmodel.LabelName(labelName)])
		if labelValue == "" {
			return nil, fmt.Errorf("label %q not found in sample metric: %v", labelName, sample.Metric)
		}
		if _, exists := result[labelValue]; exists {
			// TODO: return error
			result[labelValue] = sample
			// return nil, fmt.Errorf("duplicate label value %q found in vector", labelValue)
		}
		result[labelValue] = sample
	}
	return result, nil
}

// GroupAndSortVector Groups Prometheus Vectors by label and sort by value.
func GroupAndSortVector(vec prometheusmodel.Vector, labelName string, ascending bool) []prometheusmodel.Sample {
	sumByLabel := make(map[string]float64)

	for _, sample := range vec {
		labelValue := string(sample.Metric[prometheusmodel.LabelName(labelName)])
		if labelValue == "" {
			continue
		}
		sumByLabel[labelValue] += float64(sample.Value)
	}

	result := make([]prometheusmodel.Sample, 0, len(sumByLabel))
	for labelValue, sum := range sumByLabel {
		metric := prometheusmodel.Metric{}
		metric[prometheusmodel.LabelName(labelName)] = prometheusmodel.LabelValue(labelValue)
		result = append(result, prometheusmodel.Sample{
			Metric: metric,
			Value:  prometheusmodel.SampleValue(sum),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if ascending {
			return result[i].Value < result[j].Value
		}
		return result[i].Value > result[j].Value
	})

	return result
}

// FillMissingMatrixPoints fills missing matrix points with special values.
func FillMissingMatrixPoints(
	matrix prometheusmodel.Matrix,
	start, end time.Time,
	step time.Duration,
) prometheusmodel.Matrix {
	stepMillis := int64(step / time.Millisecond)
	// allow 1 millesecond tolerance
	tolerance := 1

	expectedTimestamps := make([]int64, 0)
	for t := start.UnixMilli(); t <= end.UnixMilli(); t += stepMillis {
		expectedTimestamps = append(expectedTimestamps, t)
	}

	result := make(prometheusmodel.Matrix, 0, len(matrix))

	for _, stream := range matrix {
		pointsMap := make(map[int64]*prometheusmodel.SamplePair)
		for i := range stream.Values {
			ts := stream.Values[i].Timestamp.Time().UnixMilli()
			pointsMap[ts] = &stream.Values[i]
		}

		newValues := make([]prometheusmodel.SamplePair, 0, len(expectedTimestamps))
		for _, ts := range expectedTimestamps {
			var matched bool
			for offset := -tolerance; offset <= tolerance; offset++ {
				if point, ok := pointsMap[ts+int64(offset)]; ok {
					newValues = append(newValues, *point)
					matched = true
					break
				}
			}
			if !matched {
				newValues = append(newValues, prometheusmodel.SamplePair{
					Timestamp: prometheusmodel.Time(ts),
					// TODO: find a more special value to mark missing value.
					Value: -1.0,
				})
			}
		}

		result = append(result, &prometheusmodel.SampleStream{
			Metric: stream.Metric,
			Values: newValues,
		})
	}

	return result
}
