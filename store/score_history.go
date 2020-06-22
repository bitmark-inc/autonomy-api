package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type ScoreHistory interface {
	AddScoreRecord(owner string, scope schema.ScoreRecordType, score float64, ts int64) error
	GetScoreAverage(owner string, start, end int64) (float64, error)
	GetScoreTimeSeriesData(owner string, start, end int64, granularity schema.AggregationTimeGranularity) ([]schema.Bucket, error)
}

// TODO: consider user local time
func (m *mongoDB) AddScoreRecord(owner string, recordType schema.ScoreRecordType, score float64, ts int64) error {
	c := m.client.Database(m.database).Collection(schema.ScoreHistoryCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	date := time.Unix(ts, 0).Format("2006-01-02")
	query := bson.M{"owner": owner, "type": recordType, "date": date}

	var record schema.ScoreRecord
	err := c.FindOne(ctx, query).Decode(&record)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}
	score = (record.Score*record.UpdateTimes + score) / (record.UpdateTimes + 1)

	update := bson.M{
		"$set": bson.M{
			"score": score,
			"ts":    ts,
		},
		"$inc": bson.M{"update_times": 1},
		"$setOnInsert": bson.M{
			"owner": owner,
			"type":  recordType,
			"date":  date,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err = c.UpdateOne(ctx, query, update, opts)
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

func (m *mongoDB) GetScoreTimeSeriesData(owner string, start, end int64, granularity schema.AggregationTimeGranularity) ([]schema.Bucket, error) {
	c := m.client.Database(m.database).Collection(schema.ScoreHistoryCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var dateStringLength int
	switch granularity {
	case schema.AggregationByMonth:
		dateStringLength = 7 // 2006-01
	case schema.AggregationByDay:
		dateStringLength = 10 // 2006-01-02
	}

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
			"$project": bson.M{
				"owner": 1,
				"score": 1,
				"date":  bson.M{"$substr": bson.A{"$date", 0, dateStringLength}},
			},
		},
		{
			"$group": bson.M{
				"_id": "$date",
				"score": bson.M{
					"$avg": "$score",
				},
			},
		},
		{
			"$project": bson.M{
				"_id":  0,
				"name": "$_id",
				"value": bson.M{
					"$toInt": "$score",
				},
			},
		},
	}
	cursor, err := c.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	buckets := make([]schema.Bucket, 0)
	for cursor.Next(ctx) {
		var aggItem schema.Bucket
		if err := cursor.Decode(&aggItem); err != nil {
			return nil, err
		}
		buckets = append(buckets, aggItem)
	}

	return buckets, nil
}
