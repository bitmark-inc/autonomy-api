package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/utils"
)

var (
	profileWithoutPOI = schema.Profile{
		ID:            "test-account-profile-id-no-poi",
		AccountNumber: "account-test-no-poi",
	}

	profileWithOnePointsRateing = schema.Profile{
		ID:            "test-account-profile-id-suggest-resource",
		AccountNumber: "account-test-suggest-resource",
		PointsOfInterest: []schema.ProfilePOI{
			{
				ResourceRatings: schema.ProfileRatingsMetric{
					Resources: []schema.RatingResource{
						{
							Resource: schema.Resource{
								ID:   "resource_1", // important
								Name: "resource_1",
							},
						},
						{
							Resource: schema.Resource{
								ID:   "resource_2", // not important
								Name: "resource_2",
							},
						},
						{
							Resource: schema.Resource{
								ID:   "abcdefghijklmnopqrstuvwxyz", // customized
								Name: "Anything",
							},
						},
					},
				},
			},
		},
	}
	profileWithTwoPointsRateing = schema.Profile{
		ID:            "test-account-profile-id-suggest-resource-two",
		AccountNumber: "account-test-suggest-resource-two",
		PointsOfInterest: []schema.ProfilePOI{
			{
				ResourceRatings: schema.ProfileRatingsMetric{
					Resources: []schema.RatingResource{
						{
							Resource: schema.Resource{
								ID:   "resource_1", // important
								Name: "resource_1",
							},
						},
						{
							Resource: schema.Resource{
								ID:   "resource_2", // not important
								Name: "resource_2",
							},
						},
						{
							Resource: schema.Resource{
								ID:   "abcdefghijklmnopqrstuvwxyz", // customized
								Name: "Anything",
							},
						},
					},
				},
			},
			{
				ResourceRatings: schema.ProfileRatingsMetric{
					Resources: []schema.RatingResource{
						{
							Resource: schema.Resource{
								ID:   "resource_1", // important
								Name: "resource_1",
							},
						},
						{
							Resource: schema.Resource{
								ID:   "resource_2", // not important
								Name: "resource_2",
							},
						},
						{
							Resource: schema.Resource{
								ID:   "abcdefghijklmnopqrstuvwxyz", // customized
								Name: "Anything",
							},
						},
					},
				},
			},
		},
	}
)

type SuggestionTestSuite struct {
	suite.Suite
	connURI      string
	testDBName   string
	mongoClient  *mongo.Client
	testDatabase *mongo.Database
}

func NewSuggestionTestSuite(connURI, dbName string) *SuggestionTestSuite {
	return &SuggestionTestSuite{
		connURI:    connURI,
		testDBName: dbName,
	}
}

func (s *SuggestionTestSuite) SetupSuite() {
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

	os.Setenv("TEST_I18N_DIR", "../i18n")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("test")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	utils.InitI18NBundle()

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
func (s *SuggestionTestSuite) LoadMongoDBFixtures() error {
	ctx := context.Background()

	if _, err := s.testDatabase.Collection(schema.ProfileCollection).InsertMany(ctx, []interface{}{
		profileWithoutPOI,
		profileWithOnePointsRateing,
		profileWithTwoPointsRateing,
	}); err != nil {
		return err
	}

	return nil
}

// CleanMongoDB drop the whole test mongodb
func (s *SuggestionTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

func (s *SuggestionTestSuite) SetupTest() {
	if err := LoadDefaultPOIResources("en"); err != nil {
		s.T().Fatal(err)
	}
}

func (s *SuggestionTestSuite) TestListSuggestedResource() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	list, err := store.ListSuggestedResources("test-account-profile-id-suggest-resource", "en")
	s.NoError(err)
	s.Len(list, 32)
}

// TestListSuggestedResourceWithTwoPointsRatingAndSameResources tests if duplicated resource rating
// will be deduplicated.
func (s *SuggestionTestSuite) TestListSuggestedResourceWithTwoPointsRatingAndSameResources() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	list, err := store.ListSuggestedResources("test-account-profile-id-suggest-resource-two", "en")
	s.NoError(err)
	s.Len(list, 32)
}

func (s *SuggestionTestSuite) TestListSuggestedResourceForProfileWihtoutPOI() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	list, err := store.ListSuggestedResources("test-account-profile-id-no-poi", "en")
	s.NoError(err)
	s.Len(list, 30)
}

func TestSuggestionTestSuite(t *testing.T) {
	suite.Run(t, NewSuggestionTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-suggestion"))
}
