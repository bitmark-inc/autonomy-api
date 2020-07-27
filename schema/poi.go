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
	PlaceTypes      []string           `bson:"place_types" json:"-"`
	OpeningHours    map[int]string     `bson:"opening_hours"`
	ServiceOptions  map[string]bool    `bson:"service_options"`
	UpdatedAt       int64              `bson:"updated_at" json:"-"`
	Distance        *float64           `bson:"distance,omitempty"`
	ResourceScore   *float64           `bson:"resource_score,omitempty"`
	ResourceRatings POIRatingsMetric   `bson:"resource_ratings" json:"-"`
}

type ProfilePOI struct {
	ID              primitive.ObjectID   `bson:"id" json:"id"`
	Alias           string               `bson:"alias" json:"alias"`
	Address         string               `bson:"address" json:"address"`
	OpeningHours    map[int]string       `bson:"opening_hours" json:"opening_hours"`
	ServiceOptions  map[string]bool      `bson:"service_options" json:"service_options"`
	Score           float64              `bson:"score" json:"score"`
	PlaceType       string               `bson:"place_type" json:"-"`
	Metric          Metric               `bson:"metric" json:"-"`
	ResourceRatings ProfileRatingsMetric `bson:"resource_ratings" json:"-"`
	UpdatedAt       time.Time            `bson:"updated_at" json:"-"`
	Monitored       bool                 `bson:"monitored" json:"-"`
}

type ResourceRatingInfo struct {
	Score  float64 `json:"score"`
	Counts int     `json:"counts"`
}

// POIDetail is a client response **ONLY** structure since the data come
// from both schema Profile.PointsOfInterest & POI
type POIDetail struct {
	ProfilePOI          `bson:",inline"`
	Location            *Location             `json:"location"`
	Distance            *float64              `json:"distance,omitempty"`
	ResourceScore       float64               `json:"resource_score"`
	ResourceRatingCount int64                 `json:"resource_rating_count"`
	ResourceRatings     map[string]RatingInfo `json:"resource_ratings"`
	InfoLastUpdated     int64                 `json:"info_last_updated"`
	ResourceLastUpdated int64                 `json:"rating_last_updated"`
	PlaceTypes          []string              `json:"place_types,omitempty"`
}
