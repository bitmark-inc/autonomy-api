package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateIntevalReport(t *testing.T) {
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
