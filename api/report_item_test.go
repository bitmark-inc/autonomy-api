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
	assert.Equal(t, map[string]*reportItem{
		"a": {
			Name:       "",
			Count:      3,
			ChangeRate: 100,
		},
		"b": {
			Name:       "",
			Count:      5,
			ChangeRate: 150,
		},
		"c": {
			Name:       "",
			Count:      0,
			ChangeRate: -100,
		},
	}, entries)
}
