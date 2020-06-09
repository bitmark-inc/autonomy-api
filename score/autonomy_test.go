package score

import (
	"testing"

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
