package schema

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	POICollection = "poi"
)

type POI struct {
	ID       primitive.ObjectID `bson:"_id"`
	Location *GeoJSON           `bson:"location"`
	Score    float64            `bson:"score"`
	Metric   Metric             `bson:"metric"`
}

type ProfilePOI struct {
	ID        primitive.ObjectID `bson:"id" json:"id"`
	Alias     string             `bson:"alias" json:"alias"`
	Address   string             `bson:"address" json:"address"`
	Score     float64            `bson:"score" json:"score"`
	Metric    Metric             `bson:"metric"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// POIDetail is a client response **ONLY** structure since the data come
// from both schema Profile.PointsOfInterest & POI
type POIDetail struct {
	ProfilePOI `bson:",inline"`
	Location   *Location `json:"location"`
}
