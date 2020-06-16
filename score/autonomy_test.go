package score

import (
	"testing"
	// "time"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/stretchr/testify/assert"
)

func TestCalculateIndividualAutonomyScoreIncrease(t *testing.T) {

	metric := schema.Metric{
		Score:          100,
		ScoreYesterday: 80,
	}

	individualMetric := schema.IndividualMetric{
		Score:          100,
		ScoreYesterday: 80,
	}

	score, delta := CalculateIndividualAutonomyScore(individualMetric, metric)
	assert.Equal(t, 100.0, score)
	assert.Equal(t, 25.0, delta)
}

func TestCalculateIndividualAutonomyScoreDecrease(t *testing.T) {

	metric := schema.Metric{
		Score:          50,
		ScoreYesterday: 80,
	}

	individualMetric := schema.IndividualMetric{
		Score:          50,
		ScoreYesterday: 100,
	}

	score, delta := CalculateIndividualAutonomyScore(individualMetric, metric)
	assert.Equal(t, 50.0, score)
	assert.Equal(t, -47.91666666666667, delta)
}

func TestCalculatePOIAutonomyScore(t *testing.T) {
	// neighbor := schema.Metric{
	// 	Score:          50,
	// 	ScoreYesterday: 80,
	// }

	// resources := []schema.POIResourceRating{
	// 	{
	// 		Resource:       schema.Resource{ID: "resource_1"},
	// 		SumOfScore:     45,
	// 		Score:          4.5,
	// 		Ratings:        10,
	// 		LastDayRatings: 8,
	// 		LastDayScore:   4.625,
	// 		LastUpdate:     time.Now().Unix(),
	// 	},
	// 	{
	// 		Resource:       schema.Resource{ID: "resource_2"},
	// 		SumOfScore:     30,
	// 		Score:          3.75,
	// 		Ratings:        8,
	// 		LastDayRatings: 6,
	// 		LastDayScore:   3.5,
	// 		LastUpdate:     time.Now().Unix(),
	// 	},
	// 	{
	// 		Resource:       schema.Resource{ID: "resource_3"},
	// 		SumOfScore:     0,
	// 		Score:          0,
	// 		Ratings:        1,
	// 		LastDayRatings: 0,
	// 		LastDayScore:   0,
	// 		LastUpdate:     time.Now().Unix(),
	// 	},
	// 	{
	// 		Resource:       schema.Resource{ID: "resource_4"},
	// 		SumOfScore:     0,
	// 		Score:          0,
	// 		Ratings:        0,
	// 		LastDayRatings: 0,
	// 		LastDayScore:   0,
	// 		LastUpdate:     time.Now().Unix(),
	// 	},
	// }

	// FIXME:
	// autonomy_test.go:91:
	//  Error Trace:    autonomy_test.go:91
	// 	Error:          Not equal:
	// 					expected: 73.15789473684211
	// 					actual  : 76.66666666666667
	// 	Test:           TestCalculatePOIAutonomyScore
	// autonomy_test.go:92:
	// 	Error Trace:    autonomy_test.go:92
	// 	Error:          Not equal:
	// 					expected: -11.092836257309942
	// 					actual  : -6.828703703703705
	// 	Test:           TestCalculatePOIAutonomyScore
	// score, _, delta := CalculatePOIAutonomyScore(resources, neighbor)
	// assert.Equal(t, 73.15789473684211, score)
	// assert.Equal(t, -11.092836257309942, delta)
}
