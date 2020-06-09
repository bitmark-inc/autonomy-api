package score

import (
	"github.com/bitmark-inc/autonomy-api/schema"
)

const (
	DefaultScoreV1SymptomCoefficient  = 0.25
	DefaultScoreV1BehaviorCoefficient = 0.25
	DefaultScoreV1ConfirmCoefficient  = 0.5
)

func DefaultTotalScore(symptomScore, behaviorScore, confirmedScore float64) float64 {
	return TotalScoreV1(schema.ScoreCoefficient{
		Symptoms:  DefaultScoreV1SymptomCoefficient,
		Behaviors: DefaultScoreV1BehaviorCoefficient,
		Confirms:  DefaultScoreV1ConfirmCoefficient,
	},
		symptomScore,
		behaviorScore,
		confirmedScore)
}

func TotalScoreV1(c schema.ScoreCoefficient, symptomScore, behaviorScore, confirmedScore float64) float64 {
	return c.Symptoms*symptomScore + c.Behaviors*behaviorScore + c.Confirms*confirmedScore
}

// CheckScoreColorChange check if the color of a score need to be changed.
// Currently,
// Red:     0 ~ 33
// Yellow: 34 ~ 66
// Green:  67 ~ 100
func CheckScoreColorChange(oldScore, newScore float64) bool {
	oldScoreMod := (int(oldScore) - 1) / 33
	newScoreMod := (int(newScore) - 1) / 33

	// for case score is 100, set the value to 2
	if oldScoreMod == 3 {
		oldScoreMod = 2
	}
	if newScoreMod == 3 {
		newScoreMod = 2
	}
	return oldScoreMod != newScoreMod
}

// CheckSymptomSpike check if there is a spike for symptoms distribution
// for a given metric data
func CheckSymptomSpike(yesterdayDistribution, currentDistribution schema.SymptomDistribution) []string {
	spikeSymptoms := []string{}

	for t, c1 := range currentDistribution {
		if c0, ok := yesterdayDistribution[t]; ok {
			// check if current symptoms count has 10% more than yesterday
			if float64(c1)/float64(c0) > 1.1 {
				spikeSymptoms = append(spikeSymptoms, t)
			}
		} else {
			spikeSymptoms = append(spikeSymptoms, t)
		}

	}

	return spikeSymptoms
}

// CalculateMetric will calculate, summarize and return a metric based on collected raw metrics
func CalculateMetric(rawMetrics schema.Metric, coefficient *schema.ScoreCoefficient) schema.Metric {
	metric := rawMetrics

	UpdateSymptomMetrics(&metric)
	UpdateBehaviorMetrics(&metric)
	CalculateConfirmScore(&metric)

	if coefficient != nil {
		metric.Score = TotalScoreV1(*coefficient, metric.Details.Symptoms.Score, metric.Details.Behaviors.Score, metric.Details.Confirm.Score)
		metric.ScoreYesterday = TotalScoreV1(*coefficient,
			metric.Details.Symptoms.ScoreYesterday,
			metric.Details.Behaviors.ScoreYesterday,
			metric.Details.Confirm.ScoreYesterday)
	} else {
		metric.Score = DefaultTotalScore(metric.Details.Symptoms.Score, metric.Details.Behaviors.Score, metric.Details.Confirm.Score)
		metric.ScoreYesterday = DefaultTotalScore(
			metric.Details.Symptoms.ScoreYesterday,
			metric.Details.Behaviors.ScoreYesterday,
			metric.Details.Confirm.ScoreYesterday)
	}

	return metric
}
