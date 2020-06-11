package score

import (
	"github.com/bitmark-inc/autonomy-api/schema"
)

func ResourceScore(count int64, sum float64, rating schema.RatingResource) (int64, float64, float64) {
	sum = sum + rating.Score
	count = count + 1
	average := sum / float64(count)
	return count, sum, average
}
