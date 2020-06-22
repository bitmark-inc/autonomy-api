package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/autonomy-api/schema"
)

func TestGatherReportItems(t *testing.T) {
	currentDistribution := map[string]int{
		"a": 3,
		"b": 5,
	}
	previousDistribution := map[string]int{
		"b": 2,
		"c": 3,
	}

	entries := gatherReportItems(currentDistribution, previousDistribution)
	assert.Equal(t, 3, *entries["a"].Value)
	assert.Equal(t, 100.0, *entries["a"].ChangeRate)
	assert.Equal(t, 5, *entries["b"].Value)
	assert.Equal(t, 150.0, *entries["b"].ChangeRate)
	assert.Equal(t, 0, *entries["c"].Value)
	assert.Equal(t, -100.0, *entries["c"].ChangeRate)
}

func TestGatherReportItemsWithDistribution(t *testing.T) {
	currentDistribution := map[string][]schema.Bucket{
		"a": {
			schema.Bucket{Name: "2020-06-22", Value: 1},
			schema.Bucket{Name: "2020-06-23", Value: 1},
			schema.Bucket{Name: "2020-06-24", Value: 1},
		},
		"b": {
			schema.Bucket{Name: "2020-06-22", Value: 2},
			schema.Bucket{Name: "2020-06-23", Value: 2},
			schema.Bucket{Name: "2020-06-24", Value: 1},
		},
	}
	previousDistribution := map[string]int{
		"b": 2,
		"c": 3,
	}

	entries := gatherReportItemsWithDistribution(currentDistribution, previousDistribution, false)
	assert.Equal(t, 3, *entries["a"].Value)
	assert.Equal(t, 100.0, *entries["a"].ChangeRate)
	assert.Equal(t, map[string]int{
		"2020-06-22": 1,
		"2020-06-23": 1,
		"2020-06-24": 1,
	}, entries["a"].Distribution)
	assert.Equal(t, 5, *entries["b"].Value)
	assert.Equal(t, 150.0, *entries["b"].ChangeRate)
	assert.Equal(t, map[string]int{
		"2020-06-22": 2,
		"2020-06-23": 2,
		"2020-06-24": 1,
	}, entries["b"].Distribution)
	assert.Equal(t, 0, *entries["c"].Value)
	assert.Equal(t, -100.0, *entries["c"].ChangeRate)
	assert.Equal(t, map[string]int(nil), entries["c"].Distribution)

	entries = gatherReportItemsWithDistribution(currentDistribution, previousDistribution, true)
	assert.Equal(t, 1, *entries["a"].Value)
	assert.Equal(t, 100.0, *entries["a"].ChangeRate)
	assert.Equal(t, map[string]int{
		"2020-06-22": 1,
		"2020-06-23": 1,
		"2020-06-24": 1,
	}, entries["a"].Distribution)
	assert.Equal(t, 1, *entries["b"].Value)
	assert.Equal(t, -50.0, *entries["b"].ChangeRate)
	assert.Equal(t, map[string]int{
		"2020-06-22": 2,
		"2020-06-23": 2,
		"2020-06-24": 1,
	}, entries["b"].Distribution)
	assert.Equal(t, 0, *entries["c"].Value)
	assert.Equal(t, -100.0, *entries["c"].ChangeRate)
	assert.Equal(t, map[string]int(nil), entries["c"].Distribution)
}
