package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
)

var (
	locationNangangTrainStation = schema.GeoJSON{
		Type:        "Point",
		Coordinates: []float64{121.605387, 25.052616},
	}
	locationSinica = schema.GeoJSON{
		Type:        "Point",
		Coordinates: []float64{121.616002, 25.042959},
	}
	locationBitmark = schema.GeoJSON{
		Type:        "Point",
		Coordinates: []float64{121.611905, 25.061037},
	}
	locationTaipeiTrainStation = schema.GeoJSON{
		Type:        "Point",
		Coordinates: []float64{121.517384, 25.047950},
	}

	tsMay25Morning = time.Date(2020, 5, 25, 9, 0, 0, 0, time.UTC).UTC().Unix()
	tsMay26Morning = time.Date(2020, 5, 26, 9, 0, 0, 0, time.UTC).UTC().Unix()
	tsMay26Evening = time.Date(2020, 5, 26, 17, 0, 0, 0, time.UTC).UTC().Unix()

	// only behavior report #2, #3, and #4 should be taken into account
	// because they are near Bitmark Taipei office and are reported during 2020 May 25
	behaviorReport1 = schema.BehaviorReportData{
		ProfileID: "userA",
		Behaviors: []schema.Behavior{
			{ID: "clean_hand"},
			{ID: "social_distancing"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: tsMay25Morning, // 05-25 17:00 (UTC+8)
	}
	behaviorReport2 = schema.BehaviorReportData{
		ProfileID: "userA",
		Behaviors: []schema.Behavior{
			{ID: "clean_hand"},
			{ID: "social_distancing"},
		},
		Location:  locationNangangTrainStation,
		Timestamp: tsMay26Morning, // 05-26 17:00 (UTC+8)
	}
	behaviorReport3 = schema.BehaviorReportData{
		ProfileID: "userA",
		Behaviors: []schema.Behavior{
			{ID: "clean_hand"},
			{ID: "social_distancing"},
		},
		Location:  locationSinica,
		Timestamp: tsMay26Evening, // 05-27 01:00 (UTC+8)
	}
	behaviorReport4 = schema.BehaviorReportData{
		ProfileID: "userB",
		Behaviors: []schema.Behavior{
			{ID: "touch_face"},
			{ID: "new_behavior"},
		},
		Location:  locationBitmark,
		Timestamp: tsMay26Morning, // 05-26 17:00 (UTC+8)
	}
	behaviorReport5 = schema.BehaviorReportData{
		ProfileID: "userB",
		Behaviors: []schema.Behavior{
			{ID: "new_behavior"},
		},
		Location:  locationTaipeiTrainStation,
		Timestamp: tsMay26Evening, // 05-27 01:00 (UTC+8)
	}
)

type BehaviorTestSuite struct {
	suite.Suite
	connURI            string
	testDBName         string
	mongoClient        *mongo.Client
	testDatabase       *mongo.Database
	neighborhoodRadius int
}

func NewBehaviorTestSuite(connURI, dbName string) *BehaviorTestSuite {
	return &BehaviorTestSuite{
		connURI:            connURI,
		testDBName:         dbName,
		neighborhoodRadius: 5000, // 5 km
	}
}

func (s *BehaviorTestSuite) SetupSuite() {
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
	if _, err := s.testDatabase.Collection(schema.BehaviorReportCollection).InsertMany(ctx, []interface{}{
		behaviorReport1,
		behaviorReport2,
		behaviorReport3,
		behaviorReport4,
		behaviorReport5,
	}); err != nil {
		s.T().Fatal(err)
	}
}

// CleanMongoDB drop the whole test mongodb
func (s *BehaviorTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

func (s *BehaviorTestSuite) TestFindBehaviorDistribution() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	start := time.Date(2020, 5, 26, 0, 0, 0, 0, time.UTC).UTC().Unix()
	end := time.Date(2020, 5, 26, 24, 0, 0, 0, time.UTC).UTC().Unix()

	distribution, err := store.FindBehaviorDistribution("",
		&schema.Location{
			Longitude: locationBitmark.Coordinates[0],
			Latitude:  locationBitmark.Coordinates[1],
		}, s.neighborhoodRadius, start, end)
	s.NoError(err)
	s.Equal(map[string]int{
		"clean_hand":        2,
		"social_distancing": 2,
		"touch_face":        1,
		"new_behavior":      1,
	}, distribution)

	distribution, err = store.FindBehaviorDistribution("userA", nil, s.neighborhoodRadius, start, end)
	s.NoError(err)
	s.Equal(map[string]int{
		"clean_hand":        2,
		"social_distancing": 2,
	}, distribution)
}

func (s *BehaviorTestSuite) TestFindNearbyBehaviorReportTimes() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	start := time.Date(2020, 5, 26, 0, 0, 0, 0, time.UTC).UTC().Unix()
	end := time.Date(2020, 5, 26, 24, 0, 0, 0, time.UTC).UTC().Unix()
	count, err := store.FindNearbyBehaviorReportTimes(
		s.neighborhoodRadius,
		schema.Location{
			Longitude: locationBitmark.Coordinates[0],
			Latitude:  locationBitmark.Coordinates[1],
		}, start, end)
	s.NoError(err)
	s.Equal(3, count)
}

func (s *BehaviorTestSuite) TestGetBehaviorCountForIndividual() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	now := time.Date(2020, 5, 25, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err := store.GetBehaviorCount("userA", nil, 0, now)
	s.NoError(err)
	s.Equal(2, todayCount)
	s.Equal(0, yesterdayCount)

	now = time.Date(2020, 5, 26, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err = store.GetBehaviorCount("userA", nil, 0, now)
	s.NoError(err)
	s.Equal(4, todayCount)
	s.Equal(2, yesterdayCount)

	now = time.Date(2020, 5, 27, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err = store.GetBehaviorCount("userA", nil, 0, now)
	s.NoError(err)
	s.Equal(0, todayCount)
	s.Equal(4, yesterdayCount)
}

func (s *BehaviorTestSuite) TestGetBehaviorCountForCommunity() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	loc := &schema.Location{
		Longitude: locationBitmark.Coordinates[0],
		Latitude:  locationBitmark.Coordinates[1],
	}

	now := time.Date(2020, 5, 25, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err := store.GetBehaviorCount("", loc, s.neighborhoodRadius, now)
	s.NoError(err)
	s.Equal(2, todayCount)
	s.Equal(0, yesterdayCount)

	now = time.Date(2020, 5, 26, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err = store.GetBehaviorCount("", loc, s.neighborhoodRadius, now)
	s.NoError(err)
	s.Equal(6, todayCount)
	s.Equal(2, yesterdayCount)

	now = time.Date(2020, 5, 27, 12, 0, 0, 0, time.UTC)
	todayCount, yesterdayCount, err = store.GetBehaviorCount("", loc, s.neighborhoodRadius, now)
	s.NoError(err)
	s.Equal(0, todayCount)
	s.Equal(6, yesterdayCount)
}

func (s *BehaviorTestSuite) TestGetNearbyReportingBehaviorsUserCount() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	now := time.Date(2020, 5, 26, 12, 0, 0, 0, time.UTC)
	count, countYesterday, err := store.GetNearbyReportingUserCount(
		schema.ReportTypeBehavior,
		s.neighborhoodRadius,
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
		schema.ReportTypeBehavior,
		s.neighborhoodRadius,
		schema.Location{
			Longitude: locationTaipeiTrainStation.Coordinates[0],
			Latitude:  locationTaipeiTrainStation.Coordinates[1],
		}, now)
	s.NoError(err)
	s.Equal(1, count)
	s.Equal(0, countYesterday)
}

func (s *BehaviorTestSuite) TestGetPersonalBehaviorTimeSeriesData() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	// the start and end time covers all inserted records for testing
	start := time.Date(2020, 5, 24, 0, 0, 0, 0, time.UTC).UTC().Unix()
	end := time.Date(2020, 5, 27, 24, 0, 0, 0, time.UTC).UTC().Unix()

	// user A with timezone in UTC
	results, err := store.GetPersonalBehaviorTimeSeriesData("userA", start, end, "+00", "day")
	s.NoError(err)
	expected := map[string][]schema.Bucket{
		"clean_hand": {
			{Name: "2020-05-25", Value: 1},
			{Name: "2020-05-26", Value: 1},
		},
		"social_distancing": {
			{Name: "2020-05-25", Value: 1},
			{Name: "2020-05-26", Value: 1},
		},
	}
	s.Equal(expected, results)

	// user A with timezone in UTC
	results, err = store.GetPersonalBehaviorTimeSeriesData("userA", start, end, "+00", "month")
	s.NoError(err)
	expected = map[string][]schema.Bucket{
		"clean_hand": {
			{Name: "2020-05", Value: 2},
		},
		"social_distancing": {
			{Name: "2020-05", Value: 2},
		},
	}
	s.Equal(expected, results)

	// user A with timezone in UTC+8
	results, err = store.GetPersonalBehaviorTimeSeriesData("userA", start, end, "+08", "day")
	s.NoError(err)
	expected = map[string][]schema.Bucket{
		"clean_hand": {
			{Name: "2020-05-25", Value: 1},
			{Name: "2020-05-26", Value: 1},
			{Name: "2020-05-27", Value: 1},
		},
		"social_distancing": {
			{Name: "2020-05-25", Value: 1},
			{Name: "2020-05-26", Value: 1},
			{Name: "2020-05-27", Value: 1},
		},
	}
	s.Equal(expected, results)

	// user B with timezone in UTC
	results, err = store.GetPersonalBehaviorTimeSeriesData("userB", start, end, "+00", "day")
	s.NoError(err)
	expected = map[string][]schema.Bucket{
		"touch_face": {
			{Name: "2020-05-26", Value: 1},
		},
		"new_behavior": {
			{Name: "2020-05-26", Value: 1},
		},
	}
	s.Equal(expected, results)

	// user B with timezone in UTC+8
	results, err = store.GetPersonalBehaviorTimeSeriesData("userB", start, end, "+08", "day")
	s.NoError(err)
	expected = map[string][]schema.Bucket{
		"touch_face": {
			{Name: "2020-05-26", Value: 1},
		},
		"new_behavior": {
			{Name: "2020-05-26", Value: 1},
			{Name: "2020-05-27", Value: 1},
		},
	}
	s.Equal(expected, results)
}

func TestBehaviorTestSuite(t *testing.T) {
	suite.Run(t, NewBehaviorTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
