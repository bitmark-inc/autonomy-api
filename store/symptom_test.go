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
			{ID: " cough"},
			{ID: " fever"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: tsMay25Morning,
	}
	symptomReport2 = schema.SymptomReportData{
		ProfileID: "userA",
		Symptoms: []schema.Symptom{
			{ID: " cough"},
			{ID: " fever"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: tsMay26Morning,
	}
	symptomReport3 = schema.SymptomReportData{
		ProfileID: "userA",
		Symptoms: []schema.Symptom{
			{ID: " cough"},
			{ID: " fever"},
		},
		Location:  locationSinica,
		Timestamp: tsMay26Evening,
	}
	symptomReport4 = schema.SymptomReportData{
		ProfileID: "userB",
		Symptoms: []schema.Symptom{
			{ID: "loss_taste_smell"},
			{ID: "new_symptom_1"},
		},
		Location:  locationBitmark,
		Timestamp: tsMay26Morning,
	}
	symptomReport5 = schema.SymptomReportData{
		ProfileID: "userB",
		Symptoms: []schema.Symptom{
			{ID: "new_symptom_2"},
		},
		Location:  locationTaipeiTrainStation,
		Timestamp: tsMay26Evening,
	}
)

type SymptomTestSuite struct {
	suite.Suite
	connURI      string
	testDBName   string
	mongoClient  *mongo.Client
	testDatabase *mongo.Database
}

func NewSymptomTestSuite(connURI, dbName string) *SymptomTestSuite {
	return &SymptomTestSuite{
		connURI:    connURI,
		testDBName: dbName,
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

func TestSymptomTestSuite(t *testing.T) {
	suite.Run(t, NewSymptomTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
