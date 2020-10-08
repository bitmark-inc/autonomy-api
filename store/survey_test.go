package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type SurveyTestSuite struct {
	suite.Suite
	connURI      string
	testDBName   string
	mongoClient  *mongo.Client
	testDatabase *mongo.Database
}

func NewSurveyTestSuite(connURI, dbName string) *SurveyTestSuite {
	return &SurveyTestSuite{
		connURI:    connURI,
		testDBName: dbName,
	}
}

func (s *SurveyTestSuite) SetupSuite() {
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

func (s *SurveyTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

func (s *SurveyTestSuite) TestCreateSurvey() {
	ctx := context.Background()
	store := NewMongoStore(s.mongoClient, s.testDBName)

	id, err := store.CreateSurvey(schema.Survey{
		AccountNumber: "survey-test-account",
		SurveyID:      "my id",
		Contents:      "any-thing",
	})
	s.NoError(err)
	s.IsType("", id)
	s.NotEmpty(id)

	sid, err := primitive.ObjectIDFromHex(id)
	s.NoError(err)

	count, err := s.testDatabase.Collection(schema.SurveyCollection).CountDocuments(ctx, bson.M{"_id": sid})
	s.NoError(err)
	s.Equal(int64(1), count)
}

func (s *SurveyTestSuite) TestCreateSurveyWithoutSurveyID() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	id, err := store.CreateSurvey(schema.Survey{
		AccountNumber: "survey-test-account-without-survey-id",
		SurveyID:      "",
		Contents:      "any-thing",
	})

	s.Error(err)
	s.Empty(id)
}

func TestSurveyTestSuite(t *testing.T) {
	suite.Run(t, NewSurveyTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
