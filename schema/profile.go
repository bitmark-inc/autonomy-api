package schema

import (
	"time"
)

const (
	ProfileCollection = "profile"
)

// SymptomWeights is structure for customized symptom weights
type SymptomWeights map[string]float64

var (
	DefaultSymptomWeights = SymptomWeights{
		"cough":            2,
		"breath":           1,
		"fever":            3,
		"chills":           1,
		"muscle_pain":      2,
		"throat":           1,
		"loss_taste_smell": 2,
	}
)

// ScoreCoefficient is structure for all customized weights for calculating personal score
type ScoreCoefficient struct {
	Symptoms       float64        `json:"symptoms" bson:"symptoms"`
	Behaviors      float64        `json:"behaviors" bson:"behaviors"`
	Confirms       float64        `json:"confirms" bson:"confirms"`
	UpdatedAt      time.Time      `json:"-" bson:"updated_at"`
	SymptomWeights SymptomWeights `json:"symptom_weights" bson:"symptom_weights"`
}

type NudgeType string

const (
	NudgeSymptomFollowUp                = NudgeType("symptom_follow_up")
	NudgeBehaviorOnSelfHighRiskSymptoms = NudgeType("behavior_on_high_risk")
	NudgeBehaviorOnSymptomSpikeArea     = NudgeType("behavior_on_symptom_spike")
)

type NudgeTime map[NudgeType]time.Time

// Profile - user profile data
type Profile struct {
	ID                  string            `bson:"id"`
	AccountNumber       string            `bson:"account_number"`
	Location            *GeoJSON          `bson:"location,omitempty"`
	Timezone            string            `bson:"timezone"`
	HealthScore         float64           `bson:"health_score"`
	IndividualMetric    IndividualMetric  `bson:"individual_metric"`
	Metric              Metric            `bson:"metric"`
	ScoreCoefficient    *ScoreCoefficient `bson:"score_coefficient"`
	LastNudge           NudgeTime         `bson:"last_nudge,omitempty"`
	PointsOfInterest    []ProfilePOI      `bson:"points_of_interest,omitempty"`
	CustomizedBehaviors []Behavior        `bson:"customized_behavior"`
	CustomizedSymptoms  []Symptom         `bson:"customized_symptom"`
}

// GeoJSON - mongo location format
type GeoJSON struct {
	Type        string    `bson:"type"`
	Coordinates []float64 `bson:"coordinates"`
}
