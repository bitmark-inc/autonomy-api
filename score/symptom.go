package score

import (
	"time"

	"github.com/bitmark-inc/autonomy-api/schema"
)

func UpdateSymptomMetrics(metric *schema.Metric) {
	rawData := metric.Details.Symptoms

	totalWeight := float64(0)
	for _, w := range schema.DefaultSymptomWeights {
		totalWeight += w
	}

	weightedSum := float64(0)
	officialCountToday := 0
	nonOfficialCountToday := 0
	for symptomID, cnt := range rawData.TodayData.WeightDistribution {
		weight, ok := schema.DefaultSymptomWeights[symptomID]
		if ok {
			officialCountToday += cnt
		} else {
			weight = 1
			nonOfficialCountToday += cnt
		}

		weightedSum += float64(cnt) * weight
	}
	totalCountToday := officialCountToday + nonOfficialCountToday

	maxWeightedSumToday := float64(rawData.TotalPeople)*totalWeight + float64(nonOfficialCountToday)
	score := 100.0
	if maxWeightedSumToday > 0 {
		score = 100 * (1 - weightedSum/maxWeightedSumToday)
	}

	weightedSumYesterday := float64(0)
	officialCountYesterday := 0
	nonOfficialCountYesterday := 0
	for symptomID, cnt := range rawData.YesterdayData.WeightDistribution {
		weight, ok := schema.DefaultSymptomWeights[symptomID]
		if ok {
			officialCountYesterday += cnt
		} else {
			weight = 1
			nonOfficialCountYesterday += cnt
		}

		weightedSumYesterday += float64(cnt) * weight
	}
	totalCountYesterday := officialCountYesterday + nonOfficialCountYesterday

	maxWeightedSumYesterday := float64(rawData.TotalPeopleYesterday)*totalWeight + float64(nonOfficialCountYesterday)
	scoreYesterday := 100.0
	if maxWeightedSumYesterday > 0 {
		scoreYesterday = 100 * (1 - weightedSumYesterday/maxWeightedSumYesterday)
	}

	spikeList := CheckSymptomSpike(rawData.YesterdayData.WeightDistribution, rawData.TodayData.WeightDistribution)

	metric.SymptomCount = float64(totalCountToday)
	metric.SymptomDelta = ChangeRate(float64(totalCountToday), float64(totalCountYesterday))
	metric.Details.Symptoms = schema.SymptomDetail{
		Score:                score,
		ScoreYesterday:       scoreYesterday,
		TotalPeople:          float64(rawData.TotalPeople),
		TotalPeopleYesterday: float64(rawData.TotalPeopleYesterday),
		TodayData:            metric.Details.Symptoms.TodayData,
		YesterdayData:        metric.Details.Symptoms.YesterdayData,
		LastSpikeList:        spikeList,
		LastSpikeUpdate:      time.Now().UTC(),
	}
}
