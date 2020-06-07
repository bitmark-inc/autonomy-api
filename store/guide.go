package store

import (
	"context"

	"github.com/bitmark-inc/autonomy-api/schema"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Guide interface {
	NearbyTestCenter(loc schema.Location, limit int64) ([]schema.TestCenter, error)
}

func (m mongoDB) NearbyTestCenter(loc schema.Location, limit int64) ([]schema.TestCenter, error) {
	c := m.client.Database(m.database).Collection(schema.TestCenterCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	_, err := c.CountDocuments(context.Background(), bson.M{})
	if err != nil {
		return []schema.TestCenter{}, err
	}
	filter := nearQuery(loc)
	opt := options.Find().SetLimit(limit)
	if 0 == limit { //return all
		opt = options.Find()
	}
	cur, err := c.Find(ctx, filter, opt)
	if err != nil {
		return []schema.TestCenter{}, err
	}
	results := []schema.TestCenter{}
	if err := cur.All(context.TODO(), &results); err != nil {
		log.WithField("prefix", mongoLogPrefix).Errorf("NearbyTestCenter decode all  error: %s", err)
	}
	return results, nil
}
