package schema

type POIRating struct {
	ID            string             `json:"id"`
	RatingAverage *float64           `json:"rating_avg,omitempty"`
	Ratings       map[string]float64 `json:"ratings"`
}
