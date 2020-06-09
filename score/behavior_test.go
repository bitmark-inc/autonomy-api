package score

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/autonomy-api/schema"
)

func TestUpdateBehaviorMetrics(t *testing.T) {
	metric := &schema.Metric{
		Details: schema.Details{
			Behaviors: schema.BehaviorDetail{
				ReportTimes:          50,
				ReportTimesYesterday: 25,
				TodayDistribution: map[string]int{
					"clean_hand":        20,
					"social_distancing": 10,
					"touch_face":        10,
					"wear_mask":         10,
					"covering_coughs":   10,
					"clean_surface":     10,
					"new_behavior_1":    5,
					"new_behavior_2":    5,
				},
				YesterdayDistribution: map[string]int{
					"clean_hand": 20,
				},
			},
		},
	}
	UpdateBehaviorMetrics(metric)
	assert.Equal(t, "25.81", fmt.Sprintf("%.2f", metric.Details.Behaviors.Score))
	assert.Equal(t, 13.333333333333334, metric.Details.Behaviors.ScoreYesterday)
	assert.Equal(t, 80.0, metric.BehaviorCount)
	assert.Equal(t, 300.0, metric.BehaviorDelta)
}

func TestCalculateBehaviorScoreNoReportToday(t *testing.T) {
	metric := &schema.Metric{
		Details: schema.Details{
			Behaviors: schema.BehaviorDetail{
				ReportTimes:          0,
				ReportTimesYesterday: 5,
				TodayDistribution:    map[string]int{},
				YesterdayDistribution: map[string]int{
					"clean_hand": 20,
				},
			},
		},
	}
	UpdateBehaviorMetrics(metric)
	assert.Equal(t, 0.0, metric.Details.Behaviors.Score)
	assert.Equal(t, 0.0, metric.BehaviorCount)
	assert.Equal(t, -100.0, metric.BehaviorDelta)

	metric = &schema.Metric{
		Details: schema.Details{
			Behaviors: schema.BehaviorDetail{
				ReportTimes:          100,
				ReportTimesYesterday: 50,
				TodayDistribution:    nil,
				YesterdayDistribution: map[string]int{
					"clean_hand": 20,
				},
			},
		},
	}
	UpdateBehaviorMetrics(metric)
	assert.Equal(t, 0.0, metric.Details.Behaviors.Score)
	assert.Equal(t, 6.666666666666667, metric.Details.Behaviors.ScoreYesterday)
	assert.Equal(t, 0.0, metric.BehaviorCount)
	assert.Equal(t, -100.0, metric.BehaviorDelta)
}

func TestCalculateBehaviorScoreSignificantNonOfficialBehaviors(t *testing.T) {
	metric := &schema.Metric{
		Details: schema.Details{
			Behaviors: schema.BehaviorDetail{
				ReportTimes:          10,
				ReportTimesYesterday: 5,
				TodayDistribution: map[string]int{
					"clean_hand":     5,
					"new_behavior_1": 30,
					"new_behavior_2": 20,
					"new_behavior_3": 20,
				},
				YesterdayDistribution: map[string]int{
					"clean_hand": 20,
				},
			},
		},
	}
	UpdateBehaviorMetrics(metric)
	assert.Equal(t, "53.85", fmt.Sprintf("%.2f", metric.Details.Behaviors.Score))
	assert.Equal(t, 66.66666666666667, metric.Details.Behaviors.ScoreYesterday)
	assert.Equal(t, 75.0, metric.BehaviorCount)
	assert.Equal(t, 275.0, metric.BehaviorDelta)
}
