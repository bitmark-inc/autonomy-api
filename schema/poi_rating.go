package schema

type Resource struct {
	ID        string `json:"id" bson:"id"`
	Name      string `json:"name" bson:"name"`
	Important bool   `json:"-" bson:"-"`
}

type RatingResource struct {
	Resource `json:"resource" bson:"resource"`
	Score    float64 `json:"score" bson:"score"`
}

type ProfileRatingsMetric struct {
	Resources  []RatingResource `json:"resources" bson:"resources"`
	LastUpdate int64            `json:"-" bson:"last_update"`
}

type POIResourceRating struct {
	Resource       `json:"resource" bson:"resource"`
	SumOfScore     float64 `json:"-" bson:"sum"`
	Score          float64 `json:"score" bson:"score"`
	Ratings        int64   `json:"ratings" bson:"ratings"`
	LastUpdate     int64   `json:"-" bson:"last_update"`
	LastDayScore   float64 `json:"-" bson:"last_day_score"`
	LastDayRatings int64   `json:"-" bson:"last_day_rating"`
}

type POIRatingsMetric struct {
	Resources  []POIResourceRating `json:"resources" bson:"resources,omitempty"`
	LastUpdate int64               `json:"last_update" bson:"last_update"`
}
