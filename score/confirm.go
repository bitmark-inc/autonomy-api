package score

import (
	"math"

	"github.com/bitmark-inc/autonomy-api/consts"
	"github.com/bitmark-inc/autonomy-api/schema"
)

// datasetPrependZero prepend zero data for data set that does not have enough window size
func datasetPrependZero(dataset []schema.CDSScoreDataSet) []schema.CDSScoreDataSet {
	datasetSize := len(dataset)

	if datasetSize < consts.ConfirmScoreWindowSize {
		datasetLackSize := consts.ConfirmScoreWindowSize - datasetSize
		newDataset := append(make([]schema.CDSScoreDataSet, datasetLackSize), dataset...)

		for idx := 0; idx < datasetLackSize; idx++ {
			newDataset[idx] = schema.CDSScoreDataSet{Name: dataset[0].Name, Cases: 0}
		}

		return newDataset
	}

	return dataset
}

// exponentialWeightAverage calculate the weight average with exponential
// coefficient using CDS score data set
func exponentialWeightAverage(dataset []schema.CDSScoreDataSet) float64 {
	score := float64(0)
	numerator := float64(0)
	denominator := float64(0)
	for idx, val := range dataset {
		power := (float64(idx) + 1) / 2
		numerator = numerator + math.Exp(power)*val.Cases
		denominator = denominator + math.Exp(power)*(val.Cases+1)
	}

	if denominator > 0 {
		score = 1 - numerator/denominator
	}

	return score
}

func CalculateConfirmScore(metric *schema.Metric) {
	details := &metric.Details.Confirm
	dataset := details.ContinuousData

	sizeOfConfirmData := len(dataset)
	if 0 == sizeOfConfirmData {
		metric.Details.Confirm.Score = 0
		metric.Details.Confirm.ScoreYesterday = 0
		return
	} else if sizeOfConfirmData < consts.ConfirmScoreWindowSize {
		dataset = datasetPrependZero(dataset)
		details.ContinuousData = dataset
	}

	datasetYesterday := details.ContinuousData[0 : len(details.ContinuousData)-1]
	if len(datasetYesterday) < consts.ConfirmScoreWindowSize {
		datasetYesterday = datasetPrependZero(datasetYesterday)
	}

	score := exponentialWeightAverage(dataset)
	scoreYesterday := exponentialWeightAverage(datasetYesterday)

	metric.Details.Confirm.Score = score * 100
	metric.Details.Confirm.ScoreYesterday = scoreYesterday * 100
}
