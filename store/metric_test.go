package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
)

var (
	metricTestSymptomReport1 = schema.SymptomReportData{
		ProfileID: "test-account-profile-id",
		Symptoms: []schema.Symptom{
			{ID: "cough"},
			{ID: "fever"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: time.Now().Add(-24 * time.Hour).Unix(),
	}
	metricTestSymptomReport2 = schema.SymptomReportData{
		ProfileID: "test-account-profile-id",
		Symptoms: []schema.Symptom{
			{ID: "cough"},
			{ID: "fever"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: time.Now().Add(-23 * time.Hour).Unix(),
	}
	metricTestSymptomReport3 = schema.SymptomReportData{
		ProfileID: "test-account-profile-id",
		Symptoms: []schema.Symptom{
			{ID: "cough"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: time.Now().Unix(),
	}

	metricTestBehaviorReport1 = schema.BehaviorReportData{
		ProfileID: "test-account-profile-id",
		Behaviors: []schema.Behavior{
			{ID: "clean_hand"},
			{ID: "social_distancing"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: time.Now().Add(-24 * time.Hour).Unix(),
	}
	metricTestBehaviorReport2 = schema.BehaviorReportData{
		ProfileID: "test-account-profile-id",
		Behaviors: []schema.Behavior{
			{ID: "clean_hand"},
			{ID: "social_distancing"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: time.Now().Add(-23 * time.Hour).Unix(),
	}
	metricTestBehaviorReport3 = schema.BehaviorReportData{
		ProfileID: "test-account-profile-id",
		Behaviors: []schema.Behavior{
			{ID: "clean_hand"},
			{ID: "social_distancing"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: time.Now().Unix(),
	}
)

type MetricTestSuite struct {
	suite.Suite
	connURI      string
	testDBName   string
	mongoClient  *mongo.Client
	testDatabase *mongo.Database
}

func NewMetricTestSuite(connURI, dbName string) *MetricTestSuite {
	return &MetricTestSuite{
		connURI:    connURI,
		testDBName: dbName,
	}
}

func (s *MetricTestSuite) SetupSuite() {
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
	if err := s.LoadMongoDBFixtures(); err != nil {
		s.T().Fatal(err)
	}
}

// LoadMongoDBFixtures will preload fixtures into test mongodb
func (s *MetricTestSuite) LoadMongoDBFixtures() error {
	ctx := context.Background()

	if _, err := s.testDatabase.Collection(schema.ProfileCollection).InsertMany(ctx, []interface{}{
		schema.Profile{
			ID:            "test-account-profile-id",
			AccountNumber: "account-test",
		},
		schema.Profile{
			ID:            "test-account-no-data",
			AccountNumber: "account-test-no-data",
		},
	}); err != nil {
		return err
	}

	if _, err := s.testDatabase.Collection(schema.SymptomReportCollection).InsertMany(ctx, []interface{}{
		metricTestSymptomReport1,
		metricTestSymptomReport2,
		metricTestSymptomReport3,
	}); err != nil {
		s.T().Fatal(err)
	}

	if _, err := s.testDatabase.Collection(schema.BehaviorReportCollection).InsertMany(ctx, []interface{}{
		metricTestBehaviorReport1,
		metricTestBehaviorReport2,
		metricTestBehaviorReport3,
	}); err != nil {
		s.T().Fatal(err)
	}

	return nil
}

// CleanMongoDB drop the whole test mongodb
func (s *MetricTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

// TestSyncProfileIndividualMetrics tests if the function returned values
// are identical to the values in database.
func (s *MetricTestSuite) TestSyncProfileIndividualMetrics() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	m, err := store.SyncProfileIndividualMetrics("test-account-profile-id")
	s.NoError(err)

	var profile schema.Profile
	err = s.testDatabase.Collection(schema.ProfileCollection).FindOne(context.Background(), bson.M{
		"id": "test-account-profile-id",
	}, options.FindOne().SetProjection(bson.M{"individual_metric": 1})).Decode(&profile)
	s.NoError(err)

	s.Equal(float64(0), m.Score)
	s.Equal(float64(0), m.ScoreYesterday)
	s.Equal(profile.IndividualMetric.Score, m.Score)
	s.Equal(profile.IndividualMetric.ScoreYesterday, m.ScoreYesterday)
	s.Equal(profile.IndividualMetric.SymptomCount, m.SymptomCount)
	s.Equal(profile.IndividualMetric.SymptomDelta, m.SymptomDelta)
	s.Equal(profile.IndividualMetric.BehaviorCount, m.BehaviorCount)
	s.Equal(profile.IndividualMetric.BehaviorDelta, m.BehaviorDelta)
}

// TestSyncProfileIndividualMetricsWithoutData tests if the score will be
// set to 100 if there is not symptom data
func (s *MetricTestSuite) TestSyncProfileIndividualMetricsWithoutData() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	m, err := store.SyncProfileIndividualMetrics("test-account-no-data")
	s.NoError(err)

	var profile schema.Profile
	err = s.testDatabase.Collection(schema.ProfileCollection).FindOne(context.Background(), bson.M{
		"id": "test-account-no-data",
	}, options.FindOne().SetProjection(bson.M{"individual_metric": 1})).Decode(&profile)
	s.NoError(err)

	s.Equal(float64(100), m.Score)
	s.Equal(float64(100), m.ScoreYesterday)
	s.Equal(profile.IndividualMetric.Score, m.Score)
	s.Equal(profile.IndividualMetric.ScoreYesterday, m.ScoreYesterday)
	s.Equal(profile.IndividualMetric.SymptomCount, m.SymptomCount)
	s.Equal(profile.IndividualMetric.SymptomDelta, m.SymptomDelta)
	s.Equal(profile.IndividualMetric.BehaviorCount, m.BehaviorCount)
	s.Equal(profile.IndividualMetric.BehaviorDelta, m.BehaviorDelta)
}

func TestMetricTestSuite(t *testing.T) {
	suite.Run(t, NewMetricTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
