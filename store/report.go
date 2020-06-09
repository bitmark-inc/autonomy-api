package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type Report interface {
	GetNearbyReportingUserCount(reportType schema.ReportType, dist int, loc schema.Location, now time.Time) (int, int, error)
}

func userCountPipelineByTime(dist int, loc schema.Location, startAt, EndAt time.Time) []bson.M {
	return []bson.M{
		aggStageGeoProximity(dist, loc),
		aggStageReportedBetween(startAt.Unix(), EndAt.Unix()),
		{
			"$group": bson.M{
				"_id": "$profile_id",
				"count": bson.M{
					"$sum": 1,
				},
			},
		},
		{
			"$group": bson.M{
				"_id": nil,
				"count": bson.M{
					"$sum": 1,
				},
			},
		},
	}
}

// GetNearbyReportingUserCount returns the number of users who have reported symptoms/behaviors
// in the specified area on the day of a given time and the day before that day.
func (m *mongoDB) GetNearbyReportingUserCount(reportType schema.ReportType, dist int, loc schema.Location, now time.Time) (int, int, error) {
	var c *mongo.Collection
	switch reportType {
	case schema.ReportTypeSymptom:
		c = m.client.Database(m.database).Collection(schema.SymptomReportCollection)
	case schema.ReportTypeBehavior:
		c = m.client.Database(m.database).Collection(schema.BehaviorReportCollection)
	default:
		return 0, 0, errors.New("invalid report type")
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	yesterdayStartAt, todayStartAt, tomorrowStartAt := getStartTimeOfConsecutiveDays(now)

	var todayCount, yesterdayCount int
	{
		cursor, err := c.Aggregate(ctx, userCountPipelineByTime(dist, loc, todayStartAt, tomorrowStartAt))
		if err != nil {
			return 0, 0, err
		}

		if cursor.Next(ctx) {
			var result struct {
				Count int `bson:"count"`
			}
			if err := cursor.Decode(&result); err != nil {
				return 0, 0, err
			}
			todayCount = result.Count
		}
	}

	{
		cursor, err := c.Aggregate(ctx, userCountPipelineByTime(dist, loc, yesterdayStartAt, todayStartAt))
		if err != nil {
			return 0, 0, err
		}

		if cursor.Next(ctx) {
			var result struct {
				Count int `bson:"count"`
			}
			if err := cursor.Decode(&result); err != nil {
				return 0, 0, err
			}
			yesterdayCount = result.Count
		}
	}

	return todayCount, yesterdayCount, nil
}
