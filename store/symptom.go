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

var localizedSymptoms map[string][]schema.Symptom = map[string][]schema.Symptom{}
var localizedSuggestedSymptoms map[string][]schema.Symptom = map[string][]schema.Symptom{}

type Symptom interface {
	CreateSymptom(symptom schema.Symptom) (string, error)
	ListOfficialSymptoms(string) ([]schema.Symptom, error)
	ListSuggestedSymptoms(lang string) ([]schema.Symptom, error)
	ListCustomizedSymptoms() ([]schema.Symptom, error)
	SymptomReportSave(data *schema.SymptomReportData) error
	FindSymptomsByIDs(ids []string) ([]schema.Symptom, error)
	FindNearbySymptomDistribution(dist int, loc schema.Location, start, end int64) (map[string]int, error)
	FindNearbyNonOfficialSymptoms(dist int, loc schema.Location) ([]schema.Symptom, error)
	GetSymptomCount(profileID string, loc *schema.Location, dist int, now time.Time) (int, int, error)
}

func (m *mongoDB) CreateSymptom(symptom schema.Symptom) (string, error) {
	if 0 == len(symptom.Name) {
		return "", errors.New("empty symptom")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := m.client.Database(m.database)

	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s=:=%s", symptom.Name, symptom.Desc)))

	symptom.ID = hex.EncodeToString(h.Sum(nil))
	symptom.Source = schema.CustomizedSymptom

	if _, err := c.Collection(schema.SymptomCollection).InsertOne(ctx, &symptom); err != nil {
		if we, hasErr := err.(mongo.WriteException); hasErr {
			if 1 == len(we.WriteErrors) && DuplicateKeyCode == we.WriteErrors[0].Code {
				return symptom.ID, nil
			}
		}
		return "", err
	}

	return symptom.ID, nil
}

func (m *mongoDB) ListOfficialSymptoms(lang string) ([]schema.Symptom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	lang = strings.ReplaceAll(strings.ToLower(lang), "-", "_")

	if symptoms, ok := localizedSymptoms[lang]; ok {
		return symptoms, nil
	}

	c := m.client.Database(m.database)

	query := bson.M{"source": schema.OfficialSymptom}

	cursor, err := c.Collection(schema.SymptomCollection).Find(ctx, query, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}

	loc := utils.NewLocalizer(lang)

	symptoms := make([]schema.Symptom, 0)

	for cursor.Next(ctx) {
		var s schema.Symptom
		if err := cursor.Decode(&s); err != nil {
			return nil, err
		}

		if name, err := loc.Localize(&i18n.LocalizeConfig{
			MessageID: fmt.Sprintf("symptoms.%s.name", s.ID),
		}); err == nil {
			s.Name = name
		} else {
			log.WithError(err).Warnf("can not decode name")
		}

		symptoms = append(symptoms, s)
	}

	localizedSymptoms[lang] = symptoms

	return symptoms, nil
}

func (m *mongoDB) ListSuggestedSymptoms(lang string) ([]schema.Symptom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	lang = strings.ReplaceAll(strings.ToLower(lang), "-", "_")

	if symptoms, ok := localizedSuggestedSymptoms[lang]; ok {
		return symptoms, nil
	}

	c := m.client.Database(m.database)

	query := bson.M{"source": schema.SuggestedSymptom}

	cursor, err := c.Collection(schema.SymptomCollection).Find(ctx, query, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}

	loc := utils.NewLocalizer(lang)

	symptoms := make([]schema.Symptom, 0)

	for cursor.Next(ctx) {
		var s schema.Symptom
		if err := cursor.Decode(&s); err != nil {
			return nil, err
		}

		if name, err := loc.Localize(&i18n.LocalizeConfig{
			MessageID: fmt.Sprintf("symptoms.%s.name", s.ID),
		}); err == nil {
			s.Name = name
		} else {
			log.WithError(err).Warnf("can not decode name")
		}

		symptoms = append(symptoms, s)
	}

	localizedSuggestedSymptoms[lang] = symptoms

	return symptoms, nil
}

func (m *mongoDB) ListCustomizedSymptoms() ([]schema.Symptom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database)
	query := bson.M{"source": schema.CustomizedSymptom}
	cursor, err := c.Collection(schema.SymptomCollection).Find(ctx, query, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}

	symptoms := make([]schema.Symptom, 0)
	for cursor.Next(ctx) {
		var s schema.Symptom
		if err := cursor.Decode(&s); err != nil {
			return nil, err
		}
		symptoms = append(symptoms, s)
	}

	return symptoms, nil
}

// SymptomReportSave save  a record instantly in database
func (m *mongoDB) SymptomReportSave(data *schema.SymptomReportData) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := m.client.Database(m.database)
	_, err := c.Collection(schema.SymptomReportCollection).InsertOne(ctx, *data)
	we, hasErr := err.(mongo.WriteException)
	if hasErr {
		if 1 == len(we.WriteErrors) && DuplicateKeyCode == we.WriteErrors[0].Code {
			return nil
		}
		return err
	}
	return nil
}

func (m *mongoDB) FindSymptomsByIDs(ids []string) ([]schema.Symptom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.SymptomCollection)

	query := bson.M{"_id": bson.M{"$in": ids}}

	cursor, err := c.Find(ctx, query)
	if err != nil {
		return nil, err
	}

	symptoms := make([]schema.Symptom, 0)
	for cursor.Next(ctx) {
		var s schema.Symptom
		if err := cursor.Decode(&s); err != nil {
			return nil, err
		}
		symptoms = append(symptoms, s)
	}

	return symptoms, nil
}

// FindNearbySymptomDistribution returns the mapping of each reported symptom and the number of users who have reported it
// in the specified area and within the specified time rage.
//
// Duplicated reported symptoms of a user are seen as one symptom.
//
// Here's the example: within the specified time interval, assume there are following 5 reports:
//
// | user  | symptoms              |
// |-------|-----------------------|
// | userA | [cough, fever]        |
// | userA | [fever, cough, nasal] |
// | userB | [fever]               |
// | userB | [fever]               |
// | userB | [fever] 			    |
//
// symptom_distribution = {fever: 2, cough: 1, nasal: 1}
func (m *mongoDB) FindNearbySymptomDistribution(dist int, loc schema.Location, start, end int64) (map[string]int, error) {
	c := m.client.Database(m.database).Collection(schema.SymptomReportCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	pipeline := []bson.M{
		aggStageGeoProximity(dist, loc),
		aggStageReportedBetween(start, end),
		{
			"$project": bson.M{
				"profile_id":     1,
				"account_number": 1,
				"symptoms": bson.M{
					"$concatArrays": bson.A{
						bson.M{"$ifNull": bson.A{"$official_symptoms", bson.A{}}},
						bson.M{"$ifNull": bson.A{"$customized_symptoms", bson.A{}}},
						bson.M{"$ifNull": bson.A{"$symptoms", bson.A{}}},
					},
				},
			},
		},
		{
			"$unwind": bson.M{
				"path":                       "$symptoms",
				"preserveNullAndEmptyArrays": false,
			},
		},
		{
			"$group": bson.M{
				"_id": "$profile_id",
				"symptoms": bson.M{
					"$addToSet": "$symptoms",
				},
			},
		}, // for each user, the number of types of symptoms reported
		{
			"$unwind": bson.M{
				"path":                       "$symptoms",
				"preserveNullAndEmptyArrays": false,
			},
		},
		{
			"$group": bson.M{
				"_id": "$symptoms._id",
				"count": bson.M{
					"$sum": 1,
				},
			},
		}, // for each symptom, the number of users who have reported it
	}

	cursor, err := c.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var aggItem struct {
		SymptomID string `bson:"_id"`
		Count     int    `bson:"count"`
	}
	result := make(map[string]int)
	for cursor.Next(ctx) {
		if err := cursor.Decode(&aggItem); err != nil {
			return nil, err
		}
		result[aggItem.SymptomID] = aggItem.Count
	}

	return result, nil
}

// FindNearbyNonOfficialSymptoms returns non-official symptoms reported today in the specified area.
func (m *mongoDB) FindNearbyNonOfficialSymptoms(dist int, loc schema.Location) ([]schema.Symptom, error) {
	distribution, err := m.FindNearbySymptomDistribution(dist, loc, 0, 9223372036854775807)
	if err != nil {
		return nil, err
	}

	nonOfficialSymptomIDs := make([]string, 0)
	for symptomID := range distribution {
		if !schema.OfficialSymptoms[symptomID] {
			nonOfficialSymptomIDs = append(nonOfficialSymptomIDs, symptomID)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.SymptomCollection)
	query := bson.M{"_id": bson.M{"$in": nonOfficialSymptomIDs}}
	cursor, err := c.Find(ctx, query, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	symptoms := make([]schema.Symptom, 0)
	for cursor.Next(ctx) {
		var s schema.Symptom
		if err := cursor.Decode(&s); err != nil {
			return nil, err
		}
		symptoms = append(symptoms, s)
	}

	return symptoms, nil
}

// GetSymptomCount returns the number of reported symptoms for today and yesterday.
//
// Either profileID of loc is required.
// If profileID is provided, returned values are personal metrics.
// Otherwise, if location is provided, returned values are community metrics.
//
// Duplicated reported symptoms of a user are seen as one symptom.
func (m *mongoDB) GetSymptomCount(profileID string, loc *schema.Location, dist int, now time.Time) (int, int, error) {
	c := m.client.Database(m.database).Collection(schema.SymptomReportCollection)
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
				"profile_id": 1,
				"day": bson.M{
					"$dateToString": bson.M{
						"format": "%Y-%m-%d",
						"date": bson.M{
							"$toDate": bson.M{
								"$multiply": bson.A{"$ts", 1000},
							},
						},
					},
				},
				"symptoms": bson.M{
					"$concatArrays": bson.A{
						bson.M{"$ifNull": bson.A{"$official_symptoms", bson.A{}}},
						bson.M{"$ifNull": bson.A{"$customized_symptoms", bson.A{}}},
						bson.M{"$ifNull": bson.A{"$symptoms", bson.A{}}},
					},
				},
			},
		},
		{
			"$unwind": bson.M{
				"path":                       "$symptoms",
				"preserveNullAndEmptyArrays": false,
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"profile_id": "$profile_id",
					"day":        "$day",
				},
				"symptoms": bson.M{
					"$addToSet": "$symptoms._id",
				},
			},
		},
		{
			"$group": bson.M{
				"_id": "$_id.day",
				"count": bson.M{
					"$sum": bson.M{"$size": "$symptoms"},
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
