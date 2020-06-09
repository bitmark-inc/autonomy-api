package score

import "github.com/bitmark-inc/autonomy-api/schema"

// CalculateIndividualAutonomyScore calculates autonomy score for individual
func CalculateIndividualAutonomyScore(individualMetric schema.IndividualMetric, neighborMetric schema.Metric) (float64, float64) {
	scoreToday := neighborMetric.Score*0.2 + individualMetric.Score*0.8
	scoreYesterday := neighborMetric.ScoreYesterday*0.2 + individualMetric.ScoreYesterday*0.8

	return scoreToday, ChangeRate(float64(scoreToday), float64(scoreYesterday))
}
