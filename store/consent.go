package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type Consent interface {
	RecordConsent(r schema.ConsentRecord) error
}

func (m *mongoDB) RecordConsent(r schema.ConsentRecord) error {
	c := m.client.Database(m.database).Collection(schema.ConsentRecordsCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"participant_id": r.ParticipantID}
	update := bson.M{
		"$set": bson.M{
			"consented": r.Consented,
			"ts":        time.Now(),
		},
	}
	_, err := c.UpdateOne(ctx, filter, update, opts)
	return err
}
