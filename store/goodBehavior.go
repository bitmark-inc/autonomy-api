package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/utils"
)

const (
	DuplicateKeyCode = 11000
)

var localizedBehaviors map[string][]schema.Behavior = map[string][]schema.Behavior{}

// GoodBehaviorReport save a GoodBehaviorData into Database
type GoodBehaviorReport interface {
	CreateBehavior(behavior schema.Behavior) (string, error)
	GoodBehaviorSave(data *schema.BehaviorReportData) error
	FindBehaviorsByIDs(ids []string) ([]schema.Behavior, error)
	FindBehaviorDistribution(profileID string, loc *schema.Location, dist int, start, end int64) (map[string]int, error)
	FindNearbyBehaviorReportTimes(dist int, loc schema.Location, start, end int64) (int, error)
	FindNearbyNonOfficialBehaviors(dist int, loc schema.Location) ([]schema.Behavior, error)
	ListOfficialBehavior(string) ([]schema.Behavior, error)
	ListCustomizedBehaviors() ([]schema.Behavior, error)
	GetBehaviorCount(profileID string, loc *schema.Location, dist int, now time.Time) (int, int, error)
	GetPersonalBehaviorTimeSeriesData(profileID string, start, end int64, utcOffset string, granularity schema.AggregationTimeGranularity) (map[string][]schema.Bucket, error)
}

func (m *mongoDB) ListOfficialBehavior(lang string) ([]schema.Behavior, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	lang = strings.ReplaceAll(strings.ToLower(lang), "-", "_")

	if behaviors, ok := localizedBehaviors[lang]; ok {
		return behaviors, nil
	}

	c := m.client.Database(m.database)

	query := bson.M{"source": schema.OfficialBehavior}

	cursor, err := c.Collection(schema.BehaviorCollection).Find(ctx, query, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}

	loc := utils.NewLocalizer(lang)

	behaviors := make([]schema.Behavior, 0)

	for cursor.Next(ctx) {
		var b schema.Behavior
		if err := cursor.Decode(&b); err != nil {
			return nil, err
		}

		if name, err := loc.Localize(&i18n.LocalizeConfig{
			MessageID: fmt.Sprintf("behaviors.%s.name", b.ID),
		}); err == nil {
			b.Name = name
		} else {
			log.WithError(err).Warnf("can not decode name")
		}

		if desc, err := loc.Localize(&i18n.LocalizeConfig{
			MessageID: fmt.Sprintf("behaviors.%s.desc", b.ID),
		}); err == nil {
			b.Desc = desc
		} else {
			log.WithError(err).Warnf("can not decode description")
		}

		behaviors = append(behaviors, b)
	}

	return behaviors, nil
}

func (m *mongoDB) ListCustomizedBehaviors() ([]schema.Behavior, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database)
	query := bson.M{"source": schema.CustomizedBehavior}
	cursor, err := c.Collection(schema.BehaviorCollection).Find(ctx, query, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}

	behaviors := make([]schema.Behavior, 0)
	for cursor.Next(ctx) {
		var s schema.Behavior
		if err := cursor.Decode(&s); err != nil {
			return nil, err
		}
		behaviors = append(behaviors, s)
	}

	return behaviors, nil
}

func (m *mongoDB) CreateBehavior(behavior schema.Behavior) (string, error) {
	if 0 == len(behavior.Name) {
		return "", errors.New("empty behavior")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := m.client.Database(m.database)

	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s=:=%s", behavior.Name, behavior.Desc)))
	behavior.ID = schema.GoodBehaviorType(hex.EncodeToString(h.Sum(nil)))
	behavior.Source = schema.CustomizedBehavior
	if _, err := c.Collection(schema.BehaviorCollection).InsertOne(ctx, &behavior); err != nil {
		if we, hasErr := err.(mongo.WriteException); hasErr {
			if 1 == len(we.WriteErrors) && DuplicateKeyCode == we.WriteErrors[0].Code {
				return string(behavior.ID), nil
			}
		}
		return "", err
	}
	return string(behavior.ID), nil
}

// GoodBehaviorData save a GoodBehaviorData into mongoDB
func (m *mongoDB) GoodBehaviorSave(data *schema.BehaviorReportData) error {
	if 0 == len(data.Behaviors) {
		data.Behaviors = []schema.Behavior{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := m.client.Database(m.database)
	_, err := c.Collection(schema.BehaviorReportCollection).InsertOne(ctx, *data)
	we, hasErr := err.(mongo.WriteException)
	if hasErr {
		if 1 == len(we.WriteErrors) && DuplicateKeyCode == we.WriteErrors[0].Code {
			return nil
		}
		return err
	}
	return nil
}

func (m *mongoDB) FindBehaviorsByIDs(ids []string) ([]schema.Behavior, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.BehaviorCollection)

	query := bson.M{"_id": bson.M{"$in": ids}}

	cursor, err := c.Find(ctx, query)
	if err != nil {
		return nil, err
	}

	behaviors := make([]schema.Behavior, 0)
	for cursor.Next(ctx) {
		var b schema.Behavior
		if err := cursor.Decode(&b); err != nil {
			return nil, err
		}
		behaviors = append(behaviors, b)
	}

	return behaviors, nil
}

// FindBehaviorDistribution returns the mapping of each reported behavior and the number of report times
// within the specified time rage (1) from a specified user or (2) in the specified area.
//
// Here's the example: within the specified time interval, assume there are following 5 reports:
//
// | user  | behaviors                                   |
// |-------|---------------------------------------------|
// | userA | [social_distancing, clean_hand]             |
// | userA | [clean_hand, social_distancing, touch_face] |
// | userB | [clean_hand]                                |
// | userB | [clean_hand]                                |
// | userB | [clean_hand] 		                         |
//
// behavior_distribution = {social_distancing: 2, clean_hand: 5, touch_face: 1}
func (m *mongoDB) FindBehaviorDistribution(profileID string, loc *schema.Location, dist int, start, end int64) (map[string]int, error) {
	c := m.client.Database(m.database).Collection(schema.BehaviorReportCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var filter bson.M
	switch {
	case profileID != "":
		filter = bson.M{
			"$match": bson.M{
				"profile_id": profileID,
			},
		}
	case loc != nil:
		filter = aggStageGeoProximity(dist, *loc)
	default:
		return nil, errors.New("either profile ID or location not provided")
	}

	pipeline := []bson.M{
		filter,
		aggStageReportedBetween(start, end),
		{
			"$project": bson.M{
				"profile_id":     1,
				"account_number": 1,
				"behaviors": bson.M{
					"$concatArrays": bson.A{
						bson.M{"$ifNull": bson.A{"$official_behaviors", bson.A{}}},
						bson.M{"$ifNull": bson.A{"$customized_behaviors", bson.A{}}},
						bson.M{"$ifNull": bson.A{"$behaviors", bson.A{}}},
					},
				},
			},
		},
		{
			"$unwind": bson.M{
				"path":                       "$behaviors",
				"preserveNullAndEmptyArrays": false,
			},
		},
		{
			"$group": bson.M{
				"_id": "$behaviors._id",
				"count": bson.M{
					"$sum": 1,
				},
			},
		},
	}

	cursor, err := c.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var aggItem struct {
		BehaviorID string `bson:"_id"`
		Count      int    `bson:"count"`
	}
	result := make(map[string]int)
	for cursor.Next(ctx) {
		if err := cursor.Decode(&aggItem); err != nil {
			return nil, err
		}
		result[aggItem.BehaviorID] = aggItem.Count
	}

	return result, nil
}

// FindNearbyBehaviorReportTimes returns the number of report times in the specified area and within the specified time rage.
//
// Take the same case described in the above function FindBehaviorDistribution for example,
// the result is 5.
func (m *mongoDB) FindNearbyBehaviorReportTimes(dist int, loc schema.Location, start, end int64) (int, error) {
	c := m.client.Database(m.database).Collection(schema.BehaviorReportCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	pipeline := []bson.M{
		aggStageGeoProximity(dist, loc),
		aggStageReportedBetween(start, end),
		{
			"$count": "count",
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
		Count int `bson:"count"`
	}
	if err := cursor.Decode(&result); err != nil {
		return 0, err
	}

	return result.Count, nil
}

// FindNearbyNonOfficialBehaviors returns non-official behaviors in the specified area.
func (m *mongoDB) FindNearbyNonOfficialBehaviors(dist int, loc schema.Location) ([]schema.Behavior, error) {
	distribution, err := m.FindBehaviorDistribution("", &loc, dist, 0, 9223372036854775807)
	if err != nil {
		return nil, err
	}

	nonOfficialBehaviorIDs := make([]string, 0)
	for id := range distribution {
		if _, ok := schema.OfficialBehaviorMatrix[schema.GoodBehaviorType(id)]; !ok {
			nonOfficialBehaviorIDs = append(nonOfficialBehaviorIDs, id)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.BehaviorCollection)
	query := bson.M{"_id": bson.M{"$in": nonOfficialBehaviorIDs}}
	cursor, err := c.Find(ctx, query, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	behaviors := make([]schema.Behavior, 0)
	for cursor.Next(ctx) {
		var b schema.Behavior
		if err := cursor.Decode(&b); err != nil {
			return nil, err
		}
		behaviors = append(behaviors, b)
	}

	return behaviors, nil
}

// GetBehaviorCount returns the number of reported behaviors for today and yesterday.
//
// Either profileID of loc is required.
// If profileID is provided, returned values are personal metrics.
// Otherwise, if location is provided, returned values are community metrics.
//
// Either profileID of loc is required.
func (m *mongoDB) GetBehaviorCount(profileID string, loc *schema.Location, dist int, now time.Time) (int, int, error) {
	c := m.client.Database(m.database).Collection(schema.BehaviorReportCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var filter bson.M
	switch {
	case profileID != "":
		filter = bson.M{
			"$match": bson.M{
				"profile_id": profileID,
			},
		}
	case loc != nil:
		filter = aggStageGeoProximity(dist, *loc)
	default:
		return 0, 0, errors.New("either profile ID or location not provided")
	}

	yesterdayStartAt, todayStartAt, tomorrowStartAt := getStartTimeOfConsecutiveDays(now)

	pipeline := []bson.M{
		filter,
		aggStageReportedBetween(yesterdayStartAt.Unix(), tomorrowStartAt.Unix()),
		{
			"$project": bson.M{
				"day": bson.M{
					"$dateToString": bson.M{
						"format": "%Y-%m-%d",
						"date":   bson.M{"$toDate": bson.M{"$multiply": bson.A{"$ts", 1000}}},
					},
				},
				"count": bson.M{
					"$add": bson.A{
						bson.M{"$size": bson.M{"$ifNull": bson.A{"$official_behaviors", bson.A{}}}},
						bson.M{"$size": bson.M{"$ifNull": bson.A{"$customized_behaviors", bson.A{}}}},
						bson.M{"$size": bson.M{"$ifNull": bson.A{"$behaviors", bson.A{}}}},
					},
				},
			},
		},
		{
			"$group": bson.M{
				"_id": "$day",
				"count": bson.M{
					"$sum": "$count",
				},
			},
		},
	}
	cursor, err := c.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, 0, err
	}
	var aggItem struct {
		Date  string `bson:"_id"`
		Count int    `bson:"count"`
	}
	result := make(map[string]int)
	for cursor.Next(ctx) {
		if err := cursor.Decode(&aggItem); err != nil {
			return 0, 0, err
		}
		result[aggItem.Date] = aggItem.Count
	}

	today := todayStartAt.Format("2006-01-02")
	yesterday := yesterdayStartAt.Format("2006-01-02")
	return result[today], result[yesterday], nil
}

// GetPersonalBehaviorTimeSeriesData returns the number of reported behaviors
// for each time interval (determined by `granularity`) in the specified time range (determined by `start` and `end`).
// Duplicated items in a day are counted as one.
func (m *mongoDB) GetPersonalBehaviorTimeSeriesData(profileID string, start, end int64, utcOffset string, granularity schema.AggregationTimeGranularity) (map[string][]schema.Bucket, error) {
	c := m.client.Database(m.database).Collection(schema.BehaviorReportCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var dateStringLength int
	switch granularity {
	case schema.AggregationByMonth:
		dateStringLength = 7 // 2006-01
	case schema.AggregationByDay:
		dateStringLength = 10 // 2006-01-02
	}

	pipeline := []bson.M{
		{"$match": bson.M{"profile_id": profileID}},
		aggStageReportedBetween(start, end),
		{
			"$project": bson.M{
				"profile_id": 1,
				"behaviors": bson.M{
					"$concatArrays": bson.A{
						bson.M{"$ifNull": bson.A{"$official_behaviors", bson.A{}}},
						bson.M{"$ifNull": bson.A{"$customized_behaviors", bson.A{}}},
						bson.M{"$ifNull": bson.A{"$behaviors", bson.A{}}},
					},
				},
				"date": bson.M{
					"$dateToString": bson.M{
						"format": "%Y-%m-%d",
						"date": bson.M{
							"$toDate": bson.M{
								"$multiply": bson.A{"$ts", 1000},
							},
						},
						"timezone": utcOffset,
					},
				},
			},
		},
		{
			"$unwind": bson.M{
				"path":                       "$behaviors",
				"preserveNullAndEmptyArrays": false,
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"profile_id": "$profile_id",
					"date":       "$date",
				},
				"behaviors": bson.M{
					"$addToSet": "$behaviors._id",
				},
			},
		},
		{
			"$unwind": bson.M{
				"path":                       "$behaviors",
				"preserveNullAndEmptyArrays": false,
			},
		},
		{
			"$project": bson.M{
				"_id":          0,
				"behaviors_id": "$behaviors",
				"bucket_name":  bson.M{"$substr": bson.A{"$_id.date", 0, dateStringLength}},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"behaviors_id": "$behaviors_id",
					"bucket_name":  "$bucket_name",
				},
				"count": bson.M{
					"$sum": 1,
				},
			},
		},
		{
			"$sort": bson.M{
				"_id.bucket_name": 1,
			},
		},
		{
			"$group": bson.M{
				"_id": "$_id.behaviors_id",
				"buckets": bson.M{
					"$push": bson.M{
						"name":  "$_id.bucket_name",
						"value": "$count",
					},
				},
			},
		},
	}
	cursor, err := c.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	results := make(map[string][]schema.Bucket)
	for cursor.Next(ctx) {
		var aggItem schema.BucketAggregation
		if err := cursor.Decode(&aggItem); err != nil {
			return nil, err
		}
		results[aggItem.ID] = aggItem.Buckets
	}
	return results, nil
}
