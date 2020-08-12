package schema

import "time"

const (
	ConsentRecordsCollection = "consent_records"
)

type ConsentRecord struct {
	ParticipantID string    `json:"participant_id" bson:"participant_id"`
	Consented     bool      `json:"consented" bson:"consented"`
	Timestamp     time.Time `json:"ts" bson:"ts"`
}
