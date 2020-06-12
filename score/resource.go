package score

import (
	"github.com/bitmark-inc/autonomy-api/schema"
)

func ResourceScore(poiRating schema.POIResourceRating, newRating schema.RatingResource, oldRating schema.RatingResource, update bool) (int64, float64, float64) {
	sum := poiRating.SumOfScore
	count := poiRating.Ratings
	if update {
		sum = sum + newRating.Score - oldRating.Score
	} else {
		sum = sum + newRating.Score
		count = count + 1
	}
	average := sum / float64(count)
	return count, sum, average
}
