package score

import (
	"github.com/bitmark-inc/autonomy-api/schema"
)

// CalculateIndividualAutonomyScore calculates autonomy score for individual
func CalculateIndividualAutonomyScore(individualMetric schema.IndividualMetric, neighborMetric schema.Metric) (float64, float64) {
	scoreToday := neighborMetric.Score*0.2 + individualMetric.Score*0.8
	scoreYesterday := neighborMetric.ScoreYesterday*0.2 + individualMetric.ScoreYesterday*0.8

	return scoreToday, ChangeRate(float64(scoreToday), float64(scoreYesterday))
}

func CalculatePOIAutonomyScore(resources []schema.POIResourceRating, neighbor schema.Metric) (float64, float64, float64) {
	sumOfScoreToday := float64(0)
	sumOfScoreYesterday := float64(0)
	sumOfRatingsToday := float64(0)
	sumOfRatingsYesterday := float64(0)
	scoreToday := float64(0)
	scoreYesterday := float64(0)
	for _, r := range resources {
		if r.Score > 0 { // score = 0 means not rated
			sumOfScoreToday = sumOfScoreToday + r.Score*float64(r.Ratings)
			sumOfRatingsToday = sumOfRatingsToday + float64(r.Ratings)
		}
		if r.LastDayScore != 0 {
			sumOfScoreYesterday = sumOfScoreYesterday + r.LastDayScore*float64(r.LastDayRatings)
			sumOfRatingsYesterday = sumOfRatingsYesterday + float64(r.LastDayRatings)
		}
	}

	if sumOfRatingsToday != 0 {
		scoreToday = ((sumOfScoreToday / sumOfRatingsToday) / 5) * 100
	}
	if sumOfRatingsYesterday != 0 {
		scoreYesterday = ((sumOfScoreYesterday / sumOfRatingsYesterday) / 5) * 100
	}

	poiScoreToday := 0.2*neighbor.Score + 0.8*scoreToday
	poiScoreYesterday := 0.2*neighbor.ScoreYesterday + 0.8*scoreYesterday

	return poiScoreToday, poiScoreYesterday, ChangeRate(poiScoreToday, poiScoreYesterday)
}
