package score

import (
	"math"

	"github.com/bitmark-inc/autonomy-api/schema"
)

func UpdateBehaviorMetrics(metric *schema.Metric) {
	todayTotal := 0
	yesterdayTotal := 0

	officialWeightedSum := float64(0)
	nonOfficialWeightedSum := float64(0)
	for behaviorID, cnt := range metric.Details.Behaviors.TodayDistribution {
		w, ok := schema.DefaultBehaviorWeightMatrix[schema.GoodBehaviorType(behaviorID)]
		if ok {
			officialWeightedSum += w.Weight * float64(cnt)
		} else {
			nonOfficialWeightedSum += float64(cnt)
		}

		todayTotal += cnt
	}

	maxWeightedSum := float64(metric.Details.Behaviors.ReportTimes)*schema.TotalOfficialBehaviorWeight + nonOfficialWeightedSum
	// cap weighted sum of non-official behaviors
	nonOfficialWeightedSum = math.Min(nonOfficialWeightedSum, maxWeightedSum/2)
	weightedSum := officialWeightedSum + nonOfficialWeightedSum
	if maxWeightedSum > 0 {
		metric.Details.Behaviors.Score = 100 * weightedSum / maxWeightedSum
	}

	officialWeightedSumYesterday := float64(0)
	nonOfficialWeightedSumYesterday := float64(0)
	for behaviorID, cnt := range metric.Details.Behaviors.YesterdayDistribution {
		w, ok := schema.DefaultBehaviorWeightMatrix[schema.GoodBehaviorType(behaviorID)]
		if ok {
			officialWeightedSumYesterday += w.Weight * float64(cnt)
		} else {
			nonOfficialWeightedSumYesterday += float64(cnt)
		}

		yesterdayTotal += cnt
	}

	maxWeightedSumYesterday := float64(metric.Details.Behaviors.ReportTimesYesterday)*schema.TotalOfficialBehaviorWeight + nonOfficialWeightedSumYesterday
	// cap weighted sum of non-official behaviors
	nonOfficialWeightedSumYesterday = math.Min(nonOfficialWeightedSumYesterday, maxWeightedSumYesterday/2)
	weightedSumYesterday := officialWeightedSumYesterday + nonOfficialWeightedSumYesterday

	if maxWeightedSumYesterday > 0 {
		metric.Details.Behaviors.ScoreYesterday = 100 * weightedSumYesterday / maxWeightedSumYesterday
	}

	metric.BehaviorCount = float64(todayTotal)
	metric.BehaviorDelta = ChangeRate(float64(todayTotal), float64(yesterdayTotal))
}
