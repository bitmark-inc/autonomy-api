package score

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/autonomy-api/schema"
)

func TestUpdateSymptomMetrics(t *testing.T) {
	metric := &schema.Metric{
		Details: schema.Details{
			Symptoms: schema.SymptomDetail{
				TotalPeople:          10,
				TotalPeopleYesterday: 5,
				TodayData: schema.NearestSymptomData{
					WeightDistribution: map[string]int{
						"cough":       3, // weight 2
						"fever":       7, // weight 3
						"new-symptom": 1, // weight 1
					}},
				YesterdayData: schema.NearestSymptomData{
					WeightDistribution: map[string]int{
						"cough":       1,
						"fever":       1,
						"new-symptom": 2,
					}},
			},
		},
	}
	UpdateSymptomMetrics(metric)
	assert.Equal(t, "76.86", fmt.Sprintf("%.2f", metric.Details.Symptoms.Score))
	assert.Equal(t, 88.70967741935483, metric.Details.Symptoms.ScoreYesterday)
	assert.Equal(t, 10.0, metric.Details.Symptoms.TotalPeople)
	assert.Equal(t, 5.0, metric.Details.Symptoms.TotalPeopleYesterday)
	assert.Equal(t, 11.0, metric.SymptomCount)
	assert.Equal(t, 175.0, metric.SymptomDelta)

	// the function must be idempotent
	UpdateSymptomMetrics(metric)
	assert.Equal(t, "76.86", fmt.Sprintf("%.2f", metric.Details.Symptoms.Score))
	assert.Equal(t, 88.70967741935483, metric.Details.Symptoms.ScoreYesterday)
	assert.Equal(t, 10.0, metric.Details.Symptoms.TotalPeople)
	assert.Equal(t, 5.0, metric.Details.Symptoms.TotalPeopleYesterday)
	assert.Equal(t, 11.0, metric.SymptomCount)
	assert.Equal(t, 175.0, metric.SymptomDelta)
}

func TestUpdateSymptomMetricsNoReportYesterday(t *testing.T) {
	metric := &schema.Metric{
		Details: schema.Details{
			Symptoms: schema.SymptomDetail{
				TotalPeople:          5,
				TotalPeopleYesterday: 5,
				TodayData: schema.NearestSymptomData{
					WeightDistribution: schema.SymptomDistribution{
						"fever":         10, // weight 3
						"new-symptom-1": 1,  // weight 1
						"new-symptom-2": 1,  // weight 1
					}},
			},
		},
	}
	UpdateSymptomMetrics(metric)
	assert.Equal(t, "48.39", fmt.Sprintf("%.2f", metric.Details.Symptoms.Score))
	assert.Equal(t, 100.0, metric.Details.Symptoms.ScoreYesterday)
	assert.Equal(t, 5.0, metric.Details.Symptoms.TotalPeople)
	assert.Equal(t, 5.0, metric.Details.Symptoms.TotalPeopleYesterday)
	assert.Equal(t, 12.0, metric.SymptomCount)
	assert.Equal(t, 100.0, metric.SymptomDelta)
}

func TestUpdateSymptomMetricsNoReportToday(t *testing.T) {
	metric := &schema.Metric{
		Details: schema.Details{
			Symptoms: schema.SymptomDetail{
				TotalPeople:          5,
				TotalPeopleYesterday: 5,
				YesterdayData: schema.NearestSymptomData{
					WeightDistribution: schema.SymptomDistribution{
						"nasal":         10, // weight 1
						"new-symptom-1": 1,  // weight 1
						"new-symptom-2": 1,  // weight 1
					}},
			},
		},
	}
	UpdateSymptomMetrics(metric)
	assert.Equal(t, 100.0, metric.Details.Symptoms.Score)
	assert.Equal(t, 83.33333333333334, metric.Details.Symptoms.ScoreYesterday)
	assert.Equal(t, 5.0, metric.Details.Symptoms.TotalPeople)
	assert.Equal(t, 5.0, metric.Details.Symptoms.TotalPeopleYesterday)
	assert.Equal(t, 0.0, metric.SymptomCount)
	assert.Equal(t, -100.0, metric.SymptomDelta)
}
