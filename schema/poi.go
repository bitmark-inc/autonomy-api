package schema

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	POICollection = "poi"
)

type POI struct {
	ID              primitive.ObjectID `bson:"_id"`
	Location        *GeoJSON           `bson:"location"`
	Address         string             `bson:"address"`
	Alias           string             `bson:"alias"`
	Score           float64            `bson:"autonomy_score"`
	ScoreDelta      float64            `bson:"autonomy_score_delta"`
	Metric          Metric             `bson:"metric"`
	Country         string             `bson:"country" json:"-"`
	State           string             `bson:"state" json:"-"`
	County          string             `bson:"county" json:"-"`
	PlaceType       string             `bson:"place_type" json:"-"`
	Distance        *float64           `bson:"distance,omitempty"`
	ResourceScore   *float64           `bson:"resource_score,omitempty"`
	ResourceRatings POIRatingsMetric   `bson:"resource_ratings" json:"-"`
}

type ProfilePOI struct {
	ID              primitive.ObjectID   `bson:"id" json:"id"`
	Alias           string               `bson:"alias" json:"alias"`
	Address         string               `bson:"address" json:"address"`
	Score           float64              `bson:"score" json:"score"`
	PlaceType       string               `bson:"place_type" json:"place_type"`
	Metric          Metric               `bson:"metric" json:"-"`
	ResourceRatings ProfileRatingsMetric `bson:"resource_ratings" json:"-"`
	UpdatedAt       time.Time            `bson:"updated_at" json:"updated_at"`
}

// POIDetail is a client response **ONLY** structure since the data come
// from both schema Profile.PointsOfInterest & POI
type POIDetail struct {
	ProfilePOI `bson:",inline"`
	Location   *Location `json:"location"`
}
