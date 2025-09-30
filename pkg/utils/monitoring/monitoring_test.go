package monitoring

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeSample(metric model.Metric, value float64) *model.Sample {
	return &model.Sample{
		Metric: metric,
		Value:  model.SampleValue(value),
	}
}

func TestVectorToMapByLabel_Normal(t *testing.T) {
	vec := model.Vector{
		makeSample(model.Metric{"job": "api-server"}, 1),
		makeSample(model.Metric{"job": "db"}, 2),
	}

	result, err := VectorToMapByLabel(vec, "job")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "api-server")
	assert.Contains(t, result, "db")
}

func TestVectorToMapByLabel_LabelMissing(t *testing.T) {
	vec := model.Vector{
		makeSample(model.Metric{"job": "api-server"}, 1),
		makeSample(model.Metric{}, 2), // no labels
	}

	_, err := VectorToMapByLabel(vec, "job")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "label \"job\" not found in sample metric")
}

// TODO: temperaily disabled this test.
// func TestVectorToMapByLabel_DuplicateLabel(t *testing.T) {
// 	vec := model.Vector{
// 		makeSample(model.Metric{"job": "api-server"}, 1),
// 		makeSample(model.Metric{"job": "api-server"}, 2),
// 	}

// 	_, err := VectorToMapByLabel(vec, "job")
// 	require.Error(t, err)
// 	assert.Contains(t, err.Error(), "duplicate label value \"api-server\" found in vector")
// }

func TestGroupAndSortVector_Empty(t *testing.T) {
	vec := model.Vector{}
	result := GroupAndSortVector(vec, "job", true)
	assert.Empty(t, result)
}

// func TestGroupAndSortVector_Grouping(t *testing.T) {
// 	vec := model.Vector{
// 		makeSample(model.Metric{"job": "api-server"}, 1),
// 		makeSample(model.Metric{"job": "db"}, 2),
// 		makeSample(model.Metric{"job": "api-server"}, 3),
// 	}

// 	result := GroupAndSortVector(vec, "job", false)
// 	assert.Len(t, result, 2)

// 	// Check values
// 	assert.Equal(t, "api-server", string(result[0].Metric["job"]))
// 	assert.InEpsilon(t, float64(4), float64(result[0].Value), 0.001)

// 	assert.Equal(t, "db", string(result[1].Metric["job"]))
// 	assert.InEpsilon(t, float64(2), float64(result[1].Value), 0.001)
// }

func TestGroupAndSortVector_IgnoreNoLabel(t *testing.T) {
	vec := model.Vector{
		makeSample(model.Metric{"job": "api-server"}, 1),
		makeSample(model.Metric{}, 2), // no job label
	}

	result := GroupAndSortVector(vec, "job", true)
	assert.Len(t, result, 1)
	assert.Equal(t, "api-server", string(result[0].Metric["job"]))
}

func TestGroupAndSortVector_Ascending(t *testing.T) {
	vec := model.Vector{
		makeSample(model.Metric{"job": "a"}, 3),
		makeSample(model.Metric{"job": "b"}, 1),
		makeSample(model.Metric{"job": "c"}, 2),
	}

	result := GroupAndSortVector(vec, "job", true)
	assert.Equal(t, "b", string(result[0].Metric["job"])) // 1
	assert.Equal(t, "c", string(result[1].Metric["job"])) // 2
	assert.Equal(t, "a", string(result[2].Metric["job"])) // 3
}

func TestGroupAndSortVector_Descending(t *testing.T) {
	vec := model.Vector{
		makeSample(model.Metric{"job": "a"}, 3),
		makeSample(model.Metric{"job": "b"}, 1),
		makeSample(model.Metric{"job": "c"}, 2),
	}

	result := GroupAndSortVector(vec, "job", false)
	assert.Equal(t, "a", string(result[0].Metric["job"])) // 3
	assert.Equal(t, "c", string(result[1].Metric["job"])) // 2
	assert.Equal(t, "b", string(result[2].Metric["job"])) // 1
}
