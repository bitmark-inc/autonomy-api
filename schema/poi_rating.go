package schema

var DefaultResources = map[string]string{
	"resource_1": "plant-based food",
	"resource_2": "organic food",
	"resource_3": "low glycemic index (low-GI) food",
	"resource_4": "farm-to-table food",
}

type RatingResourceSort []RatingResource
type POIResourceRatingSort []POIResourceRating

type Resource struct {
	ID        string `json:"id" bson:"id"`
	Name      string `json:"name" bson:"name"`
	Important bool   `json:"-" bson:"-"`
}

type RatingResource struct {
	Resource `json:"resource" bson:"resource"`
	Score    float64 `json:"score " bson:"score"`
}

type ProfileRatingsMetric struct {
	Resources  []RatingResource `json:"resources" bson:"resources"`
	LastUpdate int64            `json:"-" bson:"last_update"`
}

type POIResourceRating struct {
	Resource   `json:"resource" bson:"resource"`
	SumOfScore float64 `json:"-" bson:"sum"`
	Score      float64 `json:"score" bson:"score"`
	Ratings    int64   `json:"ratings" json:"ratings"`
}

type POIRatingsMetric struct {
	Resources  []POIResourceRating `json:"resources" bson:"resources,omitempty"`
	LastUpdate int64               `json:"last_update" bson:"last_update"`
}

func (r RatingResourceSort) Len() int           { return len(r) }
func (r RatingResourceSort) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r RatingResourceSort) Less(i, j int) bool { return r[i].Score < r[j].Score }

func (r POIResourceRatingSort) Len() int           { return len(r) }
func (r POIResourceRatingSort) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r POIResourceRatingSort) Less(i, j int) bool { return r[i].Score < r[j].Score }
