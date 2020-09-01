package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type FeedbackTestSuite struct {
	suite.Suite
	connURI      string
	testDBName   string
	mongoClient  *mongo.Client
	testDatabase *mongo.Database
}

func NewFeedbackTestSuite(connURI, dbName string) *FeedbackTestSuite {
	return &FeedbackTestSuite{
		connURI:    connURI,
		testDBName: dbName,
	}
}

func (s *FeedbackTestSuite) SetupSuite() {
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

func (s *FeedbackTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

func (s *FeedbackTestSuite) TestCreateFeedback() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	id, err := store.CreateFeedback(schema.Feedback{
		AccountNumber: "a12345",
		UserSatisfied: true,
		Feedback:      "a lots of feedback",
	})

	s.NoError(err)
	s.IsType("", id)
	s.NotEmpty(id)
}

func TestFeedbackTestSuite(t *testing.T) {
	suite.Run(t, NewFeedbackTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
