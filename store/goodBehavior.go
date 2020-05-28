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
	FindNearbyBehaviorDistribution(dist int, loc schema.Location, start, end int64) (map[string]int, error)
	FindNearbyBehaviorReportTimes(dist int, loc schema.Location, start, end int64) (int, error)
	FindNearbyNonOfficialBehaviors(dist int, loc schema.Location) ([]schema.Behavior, error)
	ListOfficialBehavior(string) ([]schema.Behavior, error)
	ListCustomizedBehaviors() ([]schema.Behavior, error)
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

// FindNearbyBehaviorDistribution returns the mapping of each reported behavior and the number of report times
// in the specified area and within the specified time rage.
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
func (m *mongoDB) FindNearbyBehaviorDistribution(dist int, loc schema.Location, start, end int64) (map[string]int, error) {
	c := m.client.Database(m.database).Collection(schema.BehaviorReportCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	pipeline := []bson.M{
		aggStageGeoProximity(dist, loc),
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
// Take the same case described in the above function FindNearbyBehaviorDistribution for example,
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
	distribution, err := m.FindNearbyBehaviorDistribution(dist, loc, 0, 9223372036854775807)
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

func todayStartAt() int64 {
	curTime := time.Now().UTC()
	start := time.Date(curTime.Year(), curTime.Month(), curTime.Day(), 0, 0, 0, 0, time.UTC)
	return start.Unix()
}
