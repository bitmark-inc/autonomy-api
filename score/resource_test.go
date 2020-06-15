package score

import (
	"testing"

	"github.com/bitmark-inc/autonomy-api/schema"
	"gopkg.in/go-playground/assert.v1"
)

func TestResourceScoreFirstRating(t *testing.T) {
	poiResourceARating := schema.POIResourceRating{
		Resource:       schema.Resource{ID: "resource_2"},
		SumOfScore:     30,
		Score:          3.75,
		Ratings:        8,
		LastDayRatings: 6,
		LastDayScore:   3.5,
	}
	userResourceARating := schema.RatingResource{
		Resource: schema.Resource{ID: "resource_2"},
		Score:    4,
	}

	count, sum, average := ResourceScore(poiResourceARating, userResourceARating, schema.RatingResource{}, false)
	assert.Equal(t, count, int64(9))
	assert.Equal(t, sum, float64(34))
	assert.Equal(t, average, float64(34)/float64(9))
}

func TestResourceScoreUpdateRating(t *testing.T) {
	poiResourceARating := schema.POIResourceRating{
		Resource:       schema.Resource{ID: "resource_2"},
		SumOfScore:     30,
		Score:          3.75,
		Ratings:        8,
		LastDayRatings: 6,
		LastDayScore:   3.5,
	}

	userResourceARating := schema.RatingResource{
		Resource: schema.Resource{ID: "resource_2"},
		Score:    4,
	}

	userResourceAOldRating := schema.RatingResource{
		Resource: schema.Resource{ID: "resource_2"},
		Score:    3,
	}

	count, sum, average := ResourceScore(poiResourceARating, userResourceARating, userResourceAOldRating, true)
	assert.Equal(t, count, int64(8))
	assert.Equal(t, sum, float64(31))
	assert.Equal(t, average, float64(31)/float64(8))
}
