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

type ScoreHistoryTestSuite struct {
	suite.Suite
	connURI      string
	testDBName   string
	mongoClient  *mongo.Client
	testDatabase *mongo.Database
}

func NewScoreHistoryTestSuite(connURI, dbName string) *ScoreHistoryTestSuite {
	return &ScoreHistoryTestSuite{
		connURI:    connURI,
		testDBName: dbName,
	}
}

func (s *ScoreHistoryTestSuite) SetupSuite() {
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
}

func (s *ScoreHistoryTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

func (s *ScoreHistoryTestSuite) TestAddScoreRecord() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	var record schema.ScoreRecord

	// user A: first update
	firstUpdateTime := time.Date(2020, 5, 25, 12, 12, 0, 0, time.UTC)
	err := store.AddScoreRecord("userA", schema.ScoreRecordTypeIndividual, 60.0, firstUpdateTime.Unix())
	s.NoError(err)

	query := bson.M{
		"owner": "userA",
		"date":  "2020-05-25",
	}
	err = s.testDatabase.Collection(schema.ScoreHistoryCollection).FindOne(
		context.Background(), query, options.FindOne()).Decode(&record)
	s.NoError(err)
	s.Equal(60.0, record.Score)
	s.Equal(schema.ScoreRecord{
		Owner: "userA",
		Type:  schema.ScoreRecordTypeIndividual,
		Score: 60.0,
		Date:  "2020-05-25",
	}, record)

	// user A: second update in the same day
	secondUpdateTime := time.Date(2020, 5, 25, 12, 12, 0, 0, time.UTC)
	err = store.AddScoreRecord("userA", schema.ScoreRecordTypeIndividual, 75.0, secondUpdateTime.Unix())
	s.NoError(err)
	err = s.testDatabase.Collection(schema.ScoreHistoryCollection).FindOne(
		context.Background(), query, options.FindOne()).Decode(&record)
	s.NoError(err)
	s.Equal(schema.ScoreRecord{
		Owner: "userA",
		Type:  schema.ScoreRecordTypeIndividual,
		Score: 75.0,
		Date:  "2020-05-25",
	}, record)

	// user B: first update in the same day
	secondUpdateTime = time.Date(2020, 5, 25, 12, 12, 0, 0, time.UTC)
	err = store.AddScoreRecord("userB", schema.ScoreRecordTypeIndividual, 40.0, secondUpdateTime.Unix())
	s.NoError(err)
	err = s.testDatabase.Collection(schema.ScoreHistoryCollection).FindOne(
		context.Background(), query, options.FindOne()).Decode(&record)
	s.NoError(err)
	s.Equal(schema.ScoreRecord{
		Owner: "userA",
		Type:  schema.ScoreRecordTypeIndividual,
		Score: 75.0,
		Date:  "2020-05-25",
	}, record)
	err = s.testDatabase.Collection(schema.ScoreHistoryCollection).FindOne(
		context.Background(), bson.M{
			"owner": "userB",
			"date":  "2020-05-25",
		}, options.FindOne()).Decode(&record)
	s.NoError(err)
	s.Equal(schema.ScoreRecord{
		Owner: "userB",
		Type:  schema.ScoreRecordTypeIndividual,
		Score: 40.0,
		Date:  "2020-05-25",
	}, record)
}

func (s *ScoreHistoryTestSuite) TestGetScoreAverage() {
	ctx := context.Background()
	if _, err := s.testDatabase.Collection(schema.ScoreHistoryCollection).InsertMany(ctx, []interface{}{
		schema.ScoreRecord{
			Owner: "userB",
			Type:  schema.ScoreRecordTypeIndividual,
			Score: 30.0,
			Date:  "2020-05-25",
		},
		schema.ScoreRecord{
			Owner: "userC",
			Type:  schema.ScoreRecordTypeIndividual,
			Score: 20.0,
			Date:  "2020-05-25",
		},
		schema.ScoreRecord{
			Owner: "userB",
			Type:  schema.ScoreRecordTypeIndividual,
			Score: 50.0,
			Date:  "2020-05-26",
		},
		schema.ScoreRecord{
			Owner: "userB",
			Type:  schema.ScoreRecordTypeIndividual,
			Score: 40.0,
			Date:  "2020-05-27",
		},
	}); err != nil {
		s.T().Fatal(err)
	}

	store := NewMongoStore(s.mongoClient, s.testDBName)
	start := time.Date(2020, 5, 25, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 5, 27, 0, 0, 0, 0, time.UTC)
	score, err := store.GetScoreAverage("userB", start.Unix(), end.Unix())
	s.NoError(err)
	s.Equal(score, 40.0)
}

func TestScoreHistoryTestSuite(t *testing.T) {
	suite.Run(t, NewScoreHistoryTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
