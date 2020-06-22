package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/consts"
	"github.com/bitmark-inc/autonomy-api/schema"
)

var (
	symptomReport1 = schema.SymptomReportData{
		ProfileID: "userA",
		Symptoms: []schema.Symptom{
			{ID: "cough"},
			{ID: "fever"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: tsMay25Morning, // 05-25 17:00 (UTC+8)
	}
	symptomReport2 = schema.SymptomReportData{
		ProfileID: "userA",
		Symptoms: []schema.Symptom{
			{ID: "cough"},
			{ID: "fever"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: tsMay26Morning, // 05-26 17:00 (UTC+8)
	}
	symptomReport3 = schema.SymptomReportData{
		ProfileID: "userA",
		Symptoms: []schema.Symptom{
			{ID: "cough"},
			{ID: "fever"},
		},
		Location:  locationSinica, // 05-27 01:00 (UTC+8)
		Timestamp: tsMay26Evening,
	}
	symptomReport4 = schema.SymptomReportData{
		ProfileID: "userB",
		Symptoms: []schema.Symptom{
			{ID: "loss_taste_smell"},
			{ID: "new_symptom_1"},
		},
		Location:  locationBitmark,
		Timestamp: tsMay26Morning, // 05-26 17:00 (UTC+8)
	}
	symptomReport5 = schema.SymptomReportData{
		ProfileID: "userB",
		Symptoms: []schema.Symptom{
			{ID: "new_symptom_2"},
		},
		Location:  locationTaipeiTrainStation,
		Timestamp: tsMay26Evening, // 05-27 01:00 (UTC+8)
	}
)

type SymptomTestSuite struct {
	suite.Suite
	connURI            string
	testDBName         string
	mongoClient        *mongo.Client
	testDatabase       *mongo.Database
	neighborhoodRadius int
}

func NewSymptomTestSuite(connURI, dbName string) *SymptomTestSuite {
	return &SymptomTestSuite{
		connURI:            connURI,
		testDBName:         dbName,
		neighborhoodRadius: 5000, // 5 km
	}
}

func (s *SymptomTestSuite) SetupSuite() {
	if s.connURI == "" || s.testDBName == "" {
		s.T().Fatal("invalid test suite configuration")
	}

	opts := options.Client().ApplyURI(s.connURI)
	mongoClient, err := mongo.NewClient(opts)
	if nil != err {
		s.T().Fatalf("create mongo client with error: %s", err)
	}

	if err = mongoClient.Connect(context.Background()); nil != err {
		s.T().Fatalf("connect mongo database with error: %s", err.Error())
	}

	s.mongoClient = mongoClient
	s.testDatabase = mongoClient.Database(s.testDBName)

	// make sure the test suite is run with a clean environment
	if err := s.CleanMongoDB(); err != nil {
		s.T().Fatal(err)
	}

	schema.NewMongoDBIndexer(s.connURI, s.testDBName).IndexAll()

	ctx := context.Background()
	if _, err := s.testDatabase.Collection(schema.SymptomReportCollection).InsertMany(ctx, []interface{}{
		symptomReport1,
		symptomReport2,
		symptomReport3,
		symptomReport4,
		symptomReport5,
	}); err != nil {
		s.T().Fatal(err)
	}
}

func (s *SymptomTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

func (s *SymptomTestSuite) TestFindSymptomDistribution() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	start := time.Date(2020, 5, 26, 0, 0, 0, 0, time.UTC).UTC().Unix()
	end := time.Date(2020, 5, 26, 24, 0, 0, 0, time.UTC).UTC().Unix()
	distribution, err := store.FindSymptomDistribution("",
		&schema.Location{
			Longitude: locationBitmark.Coordinates[0],
			Latitude:  locationBitmark.Coordinates[1],
		}, s.neighborhoodRadius, start, end, true)
	s.NoError(err)
	s.Equal(map[string]int{
		"cough":            1,
		"fever":            1,
		"loss_taste_smell": 1,
		"new_symptom_1":    1,
	}, distribution)

	distribution, err = store.FindSymptomDistribution("",
		&schema.Location{
			Longitude: locationBitmark.Coordinates[0],
			Latitude:  locationBitmark.Coordinates[1],
		}, s.neighborhoodRadius, start, end, false)
	s.NoError(err)
	s.Equal(map[string]int{
		"cough":            2,
		"fever":            2,
		"loss_taste_smell": 1,
		"new_symptom_1":    1,
	}, distribution)

	distribution, err = store.FindSymptomDistribution("userA", nil, s.neighborhoodRadius, start, end, false)
	s.NoError(err)
	s.Equal(map[string]int{
		"cough": 2,
		"fever": 2,
	}, distribution)
}

func (s *SymptomTestSuite) TestGetSymptomCountForIndividual() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	now := time.Date(2020, 5, 25, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err := store.GetSymptomCount("userA", nil, 0, now)
	s.NoError(err)
	s.Equal(2, todayCount)
	s.Equal(0, yesterdayCount)

	now = time.Date(2020, 5, 26, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err = store.GetSymptomCount("userA", nil, 0, now)
	s.NoError(err)
	s.Equal(2, todayCount)
	s.Equal(2, yesterdayCount)

	now = time.Date(2020, 5, 27, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err = store.GetSymptomCount("userA", nil, 0, now)
	s.NoError(err)
	s.Equal(0, todayCount)
	s.Equal(2, yesterdayCount)
}

func (s *SymptomTestSuite) TestGetSymptomCountForCommunity() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	loc := &schema.Location{
		Longitude: locationBitmark.Coordinates[0],
		Latitude:  locationBitmark.Coordinates[1],
	}
	dist := consts.CORHORT_DISTANCE_RANGE

	now := time.Date(2020, 5, 25, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err := store.GetSymptomCount("", loc, dist, now)
	s.NoError(err)
	s.Equal(2, todayCount)
	s.Equal(0, yesterdayCount)

	now = time.Date(2020, 5, 26, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err = store.GetSymptomCount("", loc, dist, now)
	s.NoError(err)
	s.Equal(4, todayCount)
	s.Equal(2, yesterdayCount)

	now = time.Date(2020, 5, 27, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err = store.GetSymptomCount("", loc, dist, now)
	s.NoError(err)
	s.Equal(0, todayCount)
	s.Equal(4, yesterdayCount)
}

func (s *SymptomTestSuite) TestGetNearbyReportingSymptomsUserCount() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	dist := consts.CORHORT_DISTANCE_RANGE

	now := time.Date(2020, 5, 26, 12, 0, 0, 0, time.UTC)
	count, countYesterday, err := store.GetNearbyReportingUserCount(
		schema.ReportTypeSymptom,
		dist,
		schema.Location{
			Longitude: locationBitmark.Coordinates[0],
			Latitude:  locationBitmark.Coordinates[1],
		},
		now)
	s.NoError(err)
	s.Equal(2, count)
	s.Equal(1, countYesterday)

	now = time.Date(2020, 5, 26, 12, 0, 0, 0, time.UTC)
	count, countYesterday, err = store.GetNearbyReportingUserCount(
		schema.ReportTypeSymptom,
		dist,
		schema.Location{
			Longitude: locationTaipeiTrainStation.Coordinates[0],
			Latitude:  locationTaipeiTrainStation.Coordinates[1],
		}, now)
	s.NoError(err)
	s.Equal(1, count)
	s.Equal(0, countYesterday)
}

func (s *SymptomTestSuite) TestGetPersonalSymptomTimeSeriesData() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	// the start and end time covers all inserted records for testing
	start := time.Date(2020, 5, 24, 0, 0, 0, 0, time.UTC).UTC().Unix()
	end := time.Date(2020, 5, 27, 24, 0, 0, 0, time.UTC).UTC().Unix()

	// user A with timezone in UTC
	results, err := store.GetPersonalSymptomTimeSeriesData("userA", start, end, "+00", schema.AggregationByDay)
	s.NoError(err)
	expected := map[string][]schema.Bucket{
		"cough": {
			{Name: "2020-05-25", Value: 1},
			{Name: "2020-05-26", Value: 1},
		},
		"fever": {
			{Name: "2020-05-25", Value: 1},
			{Name: "2020-05-26", Value: 1},
		},
	}
	s.Equal(expected, results)

	// user A with timezone in UTC
	results, err = store.GetPersonalSymptomTimeSeriesData("userA", start, end, "+00", schema.AggregationByMonth)
	s.NoError(err)
	expected = map[string][]schema.Bucket{
		"cough": {
			{Name: "2020-05", Value: 2},
		},
		"fever": {
			{Name: "2020-05", Value: 2},
		},
	}
	s.Equal(expected, results)

	// user A with timezone in UTC+8
	results, err = store.GetPersonalSymptomTimeSeriesData("userA", start, end, "+08", schema.AggregationByDay)
	s.NoError(err)
	expected = map[string][]schema.Bucket{
		"cough": {
			{Name: "2020-05-25", Value: 1},
			{Name: "2020-05-26", Value: 1},
			{Name: "2020-05-27", Value: 1},
		},
		"fever": {
			{Name: "2020-05-25", Value: 1},
			{Name: "2020-05-26", Value: 1},
			{Name: "2020-05-27", Value: 1},
		},
	}
	s.Equal(expected, results)

	// user B with timezone in UTC
	results, err = store.GetPersonalSymptomTimeSeriesData("userB", start, end, "+00", schema.AggregationByDay)
	s.NoError(err)
	expected = map[string][]schema.Bucket{
		"loss_taste_smell": {
			{Name: "2020-05-26", Value: 1},
		},
		"new_symptom_1": {
			{Name: "2020-05-26", Value: 1},
		},
		"new_symptom_2": {
			{Name: "2020-05-26", Value: 1},
		},
	}
	s.Equal(expected, results)

	// user B with timezone in UTC+8
	results, err = store.GetPersonalSymptomTimeSeriesData("userB", start, end, "+08", schema.AggregationByDay)
	s.NoError(err)
	expected = map[string][]schema.Bucket{
		"loss_taste_smell": {
			{Name: "2020-05-26", Value: 1},
		},
		"new_symptom_1": {
			{Name: "2020-05-26", Value: 1},
		},
		"new_symptom_2": {
			{Name: "2020-05-27", Value: 1},
		},
	}
	s.Equal(expected, results)
}

func TestSymptomTestSuite(t *testing.T) {
	suite.Run(t, NewSymptomTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
