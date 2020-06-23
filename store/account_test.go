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

type AccountTestSuite struct {
	suite.Suite
	connURI      string
	testDBName   string
	mongoClient  *mongo.Client
	testDatabase *mongo.Database
}

func NewAccountTestSuite(connURI, dbName string) *AccountTestSuite {
	return &AccountTestSuite{
		connURI:    connURI,
		testDBName: dbName,
	}
}

func (s *AccountTestSuite) SetupSuite() {
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
func (s *AccountTestSuite) LoadMongoDBFixtures() error {
	ctx := context.Background()

	if _, err := s.testDatabase.Collection(schema.ProfileCollection).InsertMany(ctx, []interface{}{
		schema.Profile{
			ID:            "test-account-profile-id",
			AccountNumber: "account-test",
		},
	}); err != nil {
		return err
	}

	return nil
}

// CleanMongoDB drop the whole test mongodb
func (s *AccountTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

func (s *AccountTestSuite) TestAppendPOIToAccountProfile() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	accountNumber := "account-test"
	firstPoiID := primitive.NewObjectID()
	firstPoi := schema.ProfilePOI{
		ID:      firstPoiID,
		Alias:   "Bitmark Inc.",
		Address: "489-1",
	}
	secondPoiID := primitive.NewObjectID()
	secondPoi := schema.ProfilePOI{
		ID:      secondPoiID,
		Alias:   "Meeting room",
		Address: "491-1",
	}

	// add the first one
	err := store.AppendPOIToAccountProfile(accountNumber, firstPoi)
	s.NoError(err)
	count, err := s.testDatabase.Collection(schema.ProfileCollection).CountDocuments(context.Background(), bson.M{
		"account_number":           accountNumber,
		"points_of_interest.id":    firstPoiID,
		"points_of_interest.alias": firstPoi.Alias,
	})
	s.NoError(err)
	s.Equal(int64(1), count)

	// add the second one
	err = store.AppendPOIToAccountProfile(accountNumber, secondPoi)
	s.NoError(err)
	count, err = s.testDatabase.Collection(schema.ProfileCollection).CountDocuments(context.Background(), bson.M{
		"account_number":           accountNumber,
		"points_of_interest.id":    secondPoiID,
		"points_of_interest.alias": secondPoi.Alias,
	})
	s.NoError(err)
	s.Equal(int64(1), count)

	count, err = s.testDatabase.Collection(schema.ProfileCollection).CountDocuments(context.Background(), bson.M{
		"account_number":     accountNumber,
		"points_of_interest": bson.M{"$size": 2},
	})
	s.NoError(err)
	s.Equal(int64(1), count)
}

// UpdateProfileIndividualMetric tests if the IndividualMetric
// is correctly added into database
func (s *AccountTestSuite) TestUpdateProfileIndividualMetric() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	err := store.UpdateProfileIndividualMetric("test-account-profile-id", schema.IndividualMetric{
		SymptomCount:  5,
		SymptomDelta:  50,
		BehaviorCount: 5,
		BehaviorDelta: 50,
	})
	s.NoError(err)

	var profile schema.Profile
	err = s.testDatabase.Collection(schema.ProfileCollection).FindOne(context.Background(), bson.M{
		"id": "test-account-profile-id",
	}, options.FindOne().SetProjection(bson.M{"individual_metric": 1})).Decode(&profile)
	s.NoError(err)

	s.Equal(float64(5), profile.IndividualMetric.SymptomCount)
	s.Equal(float64(50), profile.IndividualMetric.SymptomDelta)
	s.Equal(float64(5), profile.IndividualMetric.BehaviorCount)
	s.Equal(float64(50), profile.IndividualMetric.BehaviorDelta)
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, NewAccountTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
