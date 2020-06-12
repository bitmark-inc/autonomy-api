package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type ScoreHistory interface {
	AddScoreRecord(owner string, scope schema.ScoreRecordType, score float64, ts int64) error
	GetScoreAverage(owner string, start, end int64) (float64, error)
}

func (m *mongoDB) AddScoreRecord(owner string, recordType schema.ScoreRecordType, score float64, ts int64) error {
	c := m.client.Database(m.database).Collection(schema.ScoreHistoryCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	date := time.Unix(ts, 0).Format("2006-01-02")
	query := bson.M{"type": recordType, "date": date}
	update := bson.M{
		"$set": bson.M{
			"score": score,
			"ts":    ts,
		},
		"$setOnInsert": bson.M{
			"owner": owner,
			"type":  recordType,
			"date":  date,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := c.UpdateOne(ctx, query, update, opts)
	return err
}

func (m *mongoDB) GetScoreAverage(owner string, start, end int64) (float64, error) {
	c := m.client.Database(m.database).Collection(schema.ScoreHistoryCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	startDate := time.Unix(start, 0).Format("2006-01-02")
	endDate := time.Unix(end, 0).Format("2006-01-02")
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"owner": owner,
				"date":  bson.M{"$gte": startDate, "$lte": endDate},
			},
		},
		{
			"$group": bson.M{
				"_id": "$owner",
				"avg": bson.M{
					"$avg": "$score",
				},
			},
		},
	}

	cursor, err := c.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	if !cursor.Next(ctx) {
		return 0, nil
	}

	var result struct {
		Avg float64 `bson:"avg"`
	}
	if err := cursor.Decode(&result); err != nil {
		return 0, err
	}

	return result.Avg, nil
}
