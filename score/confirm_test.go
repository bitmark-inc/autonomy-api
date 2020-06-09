package score

import (
	"testing"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/stretchr/testify/assert"
)

func TestCalculateConfirmScoreWithoutAnyScoreData(t *testing.T) {
	var testMetric = &schema.Metric{
		Details: schema.Details{
			Confirm: schema.ConfirmDetail{
				ContinuousData: nil,
			},
		},
	}
	CalculateConfirmScore(testMetric)
	assert.Equal(t, float64(0), testMetric.Details.Confirm.ScoreYesterday)
	assert.Equal(t, float64(0), testMetric.Details.Confirm.Score)
}

func TestCalculateConfirmScore(t *testing.T) {
	var testMetric = &schema.Metric{
		Details: schema.Details{
			Confirm: schema.ConfirmDetail{
				ContinuousData: []schema.CDSScoreDataSet{
					{"Taiwan", 1}, {"Taiwan", 2}, {"Taiwan", 3}, {"Taiwan", 4},
					{"Taiwan", 5}, {"Taiwan", 6}, {"Taiwan", 7}, {"Taiwan", 8},
					{"Taiwan", 9}, {"Taiwan", 10}, {"Taiwan", 11}, {"Taiwan", 12},
					{"Taiwan", 13}, {"Taiwan", 14},
				},
			},
		},
	}
	CalculateConfirmScore(testMetric)
	assert.Equal(t, float64(8.018420610537158), testMetric.Details.Confirm.ScoreYesterday)
	assert.Equal(t, float64(7.42319741875116), testMetric.Details.Confirm.Score)
}

func TestCalculateConfirmScoreWithTwoHighScoreRecently(t *testing.T) {
	var testMetric = &schema.Metric{
		Details: schema.Details{
			Confirm: schema.ConfirmDetail{
				ContinuousData: []schema.CDSScoreDataSet{
					{"Taiwan", 78}, {"Taiwan", 87},
				},
			},
		},
	}
	CalculateConfirmScore(testMetric)
	assert.Equal(t, float64(3.1527222514609266), testMetric.Details.Confirm.ScoreYesterday)
	assert.Equal(t, float64(1.8554644575920598), testMetric.Details.Confirm.Score)
}

func TestCalculateConfirmScoreWithTwoHighScoreAtBeginning(t *testing.T) {
	var testMetric = &schema.Metric{
		Details: schema.Details{
			Confirm: schema.ConfirmDetail{
				ContinuousData: []schema.CDSScoreDataSet{
					{"Taiwan", 78}, {"Taiwan", 87}, {"Taiwan", 0}, {"Taiwan", 0},
					{"Taiwan", 0}, {"Taiwan", 0}, {"Taiwan", 0}, {"Taiwan", 0},
					{"Taiwan", 0}, {"Taiwan", 0}, {"Taiwan", 0}, {"Taiwan", 0},
					{"Taiwan", 0}, {"Taiwan", 0},
				},
			},
		},
	}
	CalculateConfirmScore(testMetric)
	assert.Equal(t, float64(82.22540024420839), testMetric.Details.Confirm.ScoreYesterday)
	assert.Equal(t, float64(88.40847697894272), testMetric.Details.Confirm.Score)
}
