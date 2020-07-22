package store

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"googlemaps.github.io/maps"

	"github.com/bitmark-inc/autonomy-api/geo"
	"github.com/bitmark-inc/autonomy-api/geo/mocks"
	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/utils"
)

var testListAllDBName = "test_list_all"

var updatePOIID = primitive.NewObjectID()
var addedPOIID = primitive.NewObjectID()
var addedPOIID2 = primitive.NewObjectID()
var existedPOIID = primitive.NewObjectID()
var emptyAliasAddrPOIID = primitive.NewObjectID()
var noCountryPOIID = primitive.NewObjectID()
var metricPOIID = primitive.NewObjectID()
var noResourcesPOIID = primitive.NewObjectID()
var noResourcesPOIID2 = primitive.NewObjectID()
var duplicatedResourcesPOIID = primitive.NewObjectID()
var twoResourcesPOIID = primitive.NewObjectID()
var officialResourcesPOIID = primitive.NewObjectID()
var listResourcePOI1ID = primitive.NewObjectID()
var listResourcePOI2ID = primitive.NewObjectID()
var listPlaceTypePOIID = primitive.NewObjectID()
var listPlaceTypePOI2ID = primitive.NewObjectID()
var searchTextInAliasPOIID = primitive.NewObjectID()
var searchTextInAddressPOIID = primitive.NewObjectID()

var notFoundPOIID = primitive.NewObjectID()

var testLocation = schema.Location{
	Latitude:  40.7385105,
	Longitude: -73.98697609999999,
	AddressComponent: schema.AddressComponent{
		Country: "United States",
		State:   "New York",
		County:  "New York County",
	},
}

var (
	noCountryPOI = schema.POI{
		ID: noCountryPOIID,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{-73.98697609999999, 40.7385105},
		},
		PlaceType: "unknown",
	}

	updatePOI = schema.POI{
		ID: updatePOIID,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{119, 24},
		},
		Country:   "Taiwan",
		State:     "",
		County:    "Yilan County",
		PlaceType: "unknown",
	}

	addedPOI = schema.POI{
		ID: addedPOIID,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{120.123, 25.123},
		},
		Country:   "Taiwan",
		State:     "",
		County:    "Yilan County",
		PlaceType: "unknown",
	}

	addedPOI2 = schema.POI{
		ID: addedPOIID2,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{120.1234, 25.1234},
		},
		Country:   "Taiwan",
		State:     "",
		County:    "Taipei City",
		PlaceType: "unknown",
	}

	existedPOI = schema.POI{
		ID:      existedPOIID,
		Alias:   "existedPOI",
		Address: "address",
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{120.12, 25.12},
		},
		Country:   "Taiwan",
		State:     "",
		County:    "Yilan County",
		PlaceType: "unknown",
	}

	emptyAliasAddrPOI = schema.POI{
		ID: emptyAliasAddrPOIID,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{120.12998, 25.12998},
		},
		Country:   "Taiwan",
		State:     "",
		County:    "Yilan County",
		PlaceType: "unknown",
	}

	metricPOI = schema.POI{
		ID: metricPOIID,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{120.12345, 25.12345},
		},
		Country: "Taiwan",
		State:   "",
		County:  "Taipei City",
		Metric: schema.Metric{
			Score:      87,
			LastUpdate: time.Now().Unix(),
		},
		PlaceType: "unknown",
	}

	noResourcesPOI = schema.POI{
		ID: noResourcesPOIID,
	}

	noResourcesPOI2 = schema.POI{
		ID: noResourcesPOIID2,
	}

	twoResourcesPOI = schema.POI{
		ID: twoResourcesPOIID,
		ResourceRatings: schema.POIRatingsMetric{
			Resources: []schema.POIResourceRating{
				{Resource: schema.Resource{ID: "7b8d3ed5e4150b189fbbf8c19786915df963c4be727d8f73db6c0b2c249563d3", Name: "test1"}},
				{Resource: schema.Resource{ID: "382bf961d3ccb85ee38ba6af8329511c976ebdd66e052d8345d102d29d52bcd7", Name: "test2"}},
			},
		},
	}

	officialResourcesPOI = schema.POI{
		ID: officialResourcesPOIID,
		ResourceRatings: schema.POIRatingsMetric{
			Resources: []schema.POIResourceRating{
				{Resource: schema.Resource{ID: "resource_1", Name: "resource_1"}}, // important
				{Resource: schema.Resource{ID: "resource_2", Name: "resource_2"}}, // normal
			},
		},
	}

	duplicatedResourcesPOI = schema.POI{
		ID: duplicatedResourcesPOIID,
		ResourceRatings: schema.POIRatingsMetric{
			Resources: []schema.POIResourceRating{
				{Resource: schema.Resource{ID: "12c99e1e3afec0ce9c21d27794d21bfff496116b8ed9ba354f731d35cb0be6b5", Name: "test3"}},
			},
		},
	}

	listResourcePOI1 = schema.POI{
		ID: listResourcePOI1ID,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{119.12345, 25.12345},
		},
		ResourceRatings: schema.POIRatingsMetric{
			Resources: []schema.POIResourceRating{
				{Resource: schema.Resource{ID: "resource_no_rating", Name: "resource_no_rating"}},
				{Resource: schema.Resource{ID: "resource_11", Name: "resource_11"}, Ratings: 2},
				{Resource: schema.Resource{ID: "resource_12", Name: "resource_12"}, Ratings: 3},
			},
		},
	}

	listResourcePOI2 = schema.POI{
		ID: listResourcePOI2ID,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{119.12345, 25.12345},
		},
		ResourceRatings: schema.POIRatingsMetric{
			Resources: []schema.POIResourceRating{
				{Resource: schema.Resource{ID: "resource_no_rating", Name: "resource_no_rating"}},
				{Resource: schema.Resource{ID: "resource_11", Name: "resource_11"}, Ratings: 2},
			},
		},
	}

	listPlaceTypePOI = schema.POI{
		ID: listPlaceTypePOIID,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{118.12345, 25.12345},
		},
		PlaceTypes: []string{"test_place"},
	}

	listPlaceTypePOI2 = schema.POI{
		ID: listPlaceTypePOI2ID,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{118.12345, 25.12345},
		},
		PlaceTypes: []string{"test_place", "happy_run"},
	}

	searchTextInAddressPOI = schema.POI{
		ID:      searchTextInAddressPOIID,
		Address: "Bitmark",
		Alias:   "test only alias123",
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{117.12345, 25.12345},
		},
	}
	searchTextInAliasPOI = schema.POI{
		ID:      searchTextInAliasPOIID,
		Address: "test only address123",
		Alias:   "Bitmark",
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{117.12345, 25.12345},
		},
	}
)

var originAlias = "origin POI"

type POITestSuite struct {
	suite.Suite
	connURI             string
	testDBName          string
	mongoClient         *mongo.Client
	testDatabase        *mongo.Database
	testListAllDatabase *mongo.Database
	mockResolver        *mocks.MockLocationResolver
}

func NewPOITestSuite(connURI, dbName string) *POITestSuite {
	return &POITestSuite{
		connURI:    connURI,
		testDBName: dbName,
	}
}

func (s *POITestSuite) SetupSuite() {
	if s.connURI == "" || s.testDBName == "" {
		s.T().Fatal("invalid test suite configuration")
	}

	opts := options.Client().ApplyURI(s.connURI)
	mongoClient, err := mongo.NewClient(opts)
	if nil != err {
		s.T().Fatalf("create mongo client with error: %s", err)
	}
	ctrl := gomock.NewController(s.T())

	mockResolver := mocks.NewMockLocationResolver(ctrl)
	geo.SetLocationResolver(mockResolver)

	if err = mongoClient.Connect(context.Background()); nil != err {
		s.T().Fatalf("connect mongo database with error: %s", err.Error())
	}

	s.mockResolver = mockResolver
	s.mongoClient = mongoClient
	s.testDatabase = mongoClient.Database(s.testDBName)
	s.testListAllDatabase = mongoClient.Database(testListAllDBName)

	os.Setenv("TEST_I18N_DIR", "../i18n")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("test")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	utils.InitI18NBundle()

	// make sure the test suite is run with a clean environment
	if err := s.CleanMongoDB(); err != nil {
		s.T().Fatal(err)
	}
	schema.NewMongoDBIndexer(s.connURI, s.testDBName).IndexAll()
	if err := s.LoadMongoDBFixtures(); err != nil {
		s.T().Fatal(err)
	}
}

func (s *POITestSuite) SetupTest() {
	if err := LoadDefaultPOIResources("en"); err != nil {
		s.T().Fatal(err)
	}
}

// LoadMongoDBFixtures will preload fixtures into test mongodb
func (s *POITestSuite) LoadMongoDBFixtures() error {
	ctx := context.Background()

	if _, err := s.testDatabase.Collection(schema.ProfileCollection).InsertMany(ctx, []interface{}{
		schema.Profile{
			ID:            uuid.New().String(),
			AccountNumber: "account-test-add-poi",
			PointsOfInterest: []schema.ProfilePOI{
				{
					ID:    addedPOIID,
					Alias: originAlias,
				},
			},
		},
		schema.Profile{
			ID:            uuid.New().String(),
			AccountNumber: "account-test-one-poi",
			PointsOfInterest: []schema.ProfilePOI{
				{
					ID:        addedPOIID,
					Alias:     originAlias,
					Monitored: true,
				},
				{
					ID:        addedPOIID2,
					Alias:     originAlias,
					Monitored: false,
				},
			},
		},
		schema.Profile{
			ID:               uuid.New().String(),
			AccountNumber:    "account-test-no-poi",
			PointsOfInterest: []schema.ProfilePOI{},
		},
		schema.Profile{
			ID:            uuid.New().String(),
			AccountNumber: "account-test-poi-reorder",
			PointsOfInterest: []schema.ProfilePOI{
				{
					ID:    addedPOIID,
					Alias: originAlias,
				},
				{
					ID:    addedPOIID2,
					Alias: originAlias,
				},
			},
		},
		schema.Profile{
			ID:            uuid.New().String(),
			AccountNumber: "account-test-delete-poi",
			PointsOfInterest: []schema.ProfilePOI{
				{
					ID:        addedPOIID,
					Alias:     originAlias,
					Monitored: true,
				},
				{
					ID:        addedPOIID2,
					Alias:     originAlias,
					Monitored: true,
				},
			},
		},
		schema.Profile{
			ID:            uuid.New().String(),
			AccountNumber: "account-test-update-poi-alias",
			PointsOfInterest: []schema.ProfilePOI{
				{
					ID:    addedPOIID,
					Alias: originAlias,
				},
			},
		},
	}); err != nil {
		return err
	}

	if _, err := s.testDatabase.Collection(schema.POICollection).InsertMany(ctx, []interface{}{
		noCountryPOI,
		updatePOI,
		addedPOI,
		addedPOI2,
		existedPOI,
		emptyAliasAddrPOI,
		metricPOI,
		noResourcesPOI,
		noResourcesPOI2,
		twoResourcesPOI,
		duplicatedResourcesPOI,
		officialResourcesPOI,
		listResourcePOI1,
		listResourcePOI2,
		listPlaceTypePOI,
		listPlaceTypePOI2,
		searchTextInAddressPOI,
		searchTextInAliasPOI,
	}); err != nil {
		return err
	}

	if _, err := s.testListAllDatabase.Collection(schema.POICollection).InsertMany(ctx, []interface{}{
		addedPOI,
		addedPOI2,
	}); err != nil {
		return err
	}

	return nil
}

// CleanMongoDB drop the whole test mongodb
func (s *POITestSuite) CleanMongoDB() error {
	if err := s.testListAllDatabase.Drop(context.Background()); err != nil {
		return err
	}
	return s.testDatabase.Drop(context.Background())
}

// LoadGeocodingFixtures load prepared geocoding fixtures
func (s *POITestSuite) LoadGeocodingFixtures() ([]maps.GeocodingResult, error) {
	f, err := os.Open("fixtures/geo_result_union_square.json")
	if err != nil {
		return nil, err
	}
	var fixture struct {
		Results []maps.GeocodingResult `json:"results"`
	}

	if err := json.NewDecoder(f).Decode(&fixture); err != nil {
		return nil, err
	}
	return fixture.Results, nil
}

// TestAddPOI tests adding a new poi normally
func (s *POITestSuite) TestAddPOI() {
	ctx := context.Background()
	store := NewMongoStore(s.mongoClient, s.testDBName)

	s.mockResolver.EXPECT().
		GetPoliticalInfo(gomock.AssignableToTypeOf(schema.Location{})).
		Return(testLocation, nil)

	poi, err := store.AddPOI("test-poi", "", utils.UnknownPlace, 120.1, 25.1)
	s.NoError(err)
	s.Equal("United States", poi.Country)
	s.Equal("New York", poi.State)
	s.Equal("New York County", poi.County)
	s.Equal(utils.UnknownPlace, poi.PlaceType)
	s.Equal([]float64{120.1, 25.1}, poi.Location.Coordinates)

	count, err := s.testDatabase.Collection(schema.POICollection).CountDocuments(ctx, bson.M{"_id": poi.ID})
	s.NoError(err)
	s.Equal(int64(1), count)
}

// TestAddExistentPOI tests adding a poi where its coordinates has alreday added by other accounts
// but not in the test account
func (s *POITestSuite) TestAddExistentPOI() {
	ctx := context.Background()
	store := NewMongoStore(s.mongoClient, s.testDBName)

	s.mockResolver.EXPECT().
		GetPoliticalInfo(gomock.AssignableToTypeOf(schema.Location{})).
		Return(testLocation, nil)

	poi, err := store.AddPOI("any", "any", utils.UnknownPlace, existedPOI.Location.Coordinates[0], existedPOI.Location.Coordinates[1])
	s.NoError(err)
	s.Equal("existedPOI", poi.Alias)
	s.Equal("address", poi.Address)
	s.Equal("Taiwan", poi.Country)
	s.Equal("", poi.State)
	s.Equal("Yilan County", poi.County)
	s.Equal(utils.UnknownPlace, poi.PlaceType)
	s.Equal([]float64{existedPOI.Location.Coordinates[0], existedPOI.Location.Coordinates[1]}, poi.Location.Coordinates)

	count, err := s.testDatabase.Collection(schema.POICollection).CountDocuments(ctx, bson.M{"_id": existedPOIID})
	s.NoError(err)
	s.Equal(int64(1), count)
}

// TestAddPOIUpdateAliasIfEmpty tests adding a duplicated poi where its coordinates has alreday existed
// If the target POI's alias or address is empty, it will be replaced.
func (s *POITestSuite) TestAddPOIUpdateAliasIfEmpty() {
	ctx := context.Background()
	store := NewMongoStore(s.mongoClient, s.testDBName)

	s.mockResolver.EXPECT().
		GetPoliticalInfo(gomock.AssignableToTypeOf(schema.Location{})).
		Return(testLocation, nil)

	// use a different name to add an added poi
	poi, err := store.AddPOI("new-name", "new-address", utils.UnknownPlace, emptyAliasAddrPOI.Location.Coordinates[0], emptyAliasAddrPOI.Location.Coordinates[1])
	s.NoError(err)
	s.Equal("Taiwan", poi.Country)
	s.Equal("", poi.State)
	s.Equal("Yilan County", poi.County)
	s.Equal(utils.UnknownPlace, poi.PlaceType)
	s.Equal([]float64{emptyAliasAddrPOI.Location.Coordinates[0], emptyAliasAddrPOI.Location.Coordinates[1]}, poi.Location.Coordinates)

	var dbPOI schema.POI
	err = s.testDatabase.Collection(schema.POICollection).FindOne(ctx, bson.M{"_id": emptyAliasAddrPOIID}).Decode(&dbPOI)
	s.NoError(err)
	s.Equal("new-name", dbPOI.Alias)
	s.Equal("new-address", dbPOI.Address)
}

// TestListPOIForUserWithoutAny tests listing all POIs from db
func (s *POITestSuite) TestListPOIForUserWithoutAny() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	pois, err := store.ListPOI("account-test-no-poi")
	s.NoError(err)
	s.Len(pois, 0)
}

// TestListPOIForUserWithoutAny tests listing all POIs from db
func (s *POITestSuite) TestListPOIForUserWithOne() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	pois, err := store.ListPOI("account-test-one-poi")
	s.NoError(err)
	s.Len(pois, 1)
	poi := pois[0]

	s.Equal(originAlias, poi.Alias)
	s.Equal(addedPOI.Location.Coordinates[0], poi.Location.Longitude)
	s.Equal(addedPOI.Location.Coordinates[1], poi.Location.Latitude)
}

// TestListPOIForUserWithoutAny tests listing all POIs from db
func (s *POITestSuite) TestGetPOINormal() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	poi, err := store.GetPOI(addedPOIID)
	s.NoError(err)
	s.NotNil(poi)

	s.Equal(addedPOI.Location.Coordinates[0], poi.Location.Coordinates[0])
	s.Equal(addedPOI.Location.Coordinates[1], poi.Location.Coordinates[1])
	s.Equal("Taiwan", poi.Country)
	s.Equal("", poi.State)
	s.Equal("Yilan County", poi.County)
}

func (s *POITestSuite) TestGetPOIWithoutCountry() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	s.mockResolver.EXPECT().
		GetPoliticalInfo(gomock.AssignableToTypeOf(schema.Location{})).
		Return(testLocation, nil)

	poi, err := store.GetPOI(noCountryPOIID)
	s.NoError(err)
	s.NotNil(poi)

	s.Equal(noCountryPOI.Location.Coordinates[0], poi.Location.Coordinates[0])
	s.Equal(noCountryPOI.Location.Coordinates[1], poi.Location.Coordinates[1])
	s.Equal("United States", poi.Country)
	s.Equal("New York", poi.State)
	s.Equal("New York County", poi.County)
}

func (s *POITestSuite) TestUpdatePOIOrder() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	var profile schema.Profile
	err := s.testDatabase.Collection(schema.ProfileCollection).FindOne(context.Background(), bson.M{
		"account_number": "account-test-poi-reorder",
	}, options.FindOne().SetProjection(bson.M{"points_of_interest": 1})).Decode(&profile)
	s.NoError(err)

	s.Len(profile.PointsOfInterest, 2)
	s.Equal(addedPOIID, profile.PointsOfInterest[0].ID)
	s.Equal(addedPOIID2, profile.PointsOfInterest[1].ID)

	err = store.UpdatePOIOrder("account-test-poi-reorder", []string{addedPOIID2.Hex(), addedPOIID.Hex()})
	s.NoError(err)

	var profile2 schema.Profile
	err = s.testDatabase.Collection(schema.ProfileCollection).FindOne(context.Background(), bson.M{
		"account_number": "account-test-poi-reorder",
	}, options.FindOne().SetProjection(bson.M{"points_of_interest": 1})).Decode(&profile2)
	s.NoError(err)

	s.Len(profile2.PointsOfInterest, 2)
	s.Equal(addedPOIID2, profile2.PointsOfInterest[0].ID)
	s.Equal(addedPOIID, profile2.PointsOfInterest[1].ID)
}

func (s *POITestSuite) TestUpdatePOIOrderForNonexistentAccount() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	err := store.UpdatePOIOrder("account-not-found-test-poi", []string{addedPOIID2.Hex(), addedPOIID.Hex()})
	s.EqualError(err, ErrPOIListNotFound.Error())
}

func (s *POITestSuite) TestUpdatePOIOrderWithWrongID() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	err := store.UpdatePOIOrder("account-test-poi-reorder", []string{"12345678", "99987654"})
	s.EqualError(err, primitive.ErrInvalidHex.Error())
}

// TestUpdatePOIOrderMismatch tests if we try to re-order non-existent. There will nothing happen.
func (s *POITestSuite) TestUpdatePOIOrderMismatch() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	err := store.UpdatePOIOrder("account-test-poi-reorder", []string{addedPOIID.Hex()})
	s.NoError(err)
}

func (s *POITestSuite) TestUpdatePOIOrderForAccountWithoutAnyPOI() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	err := store.UpdatePOIOrder("account-test-no-poi", []string{addedPOIID.Hex()})
	s.EqualError(err, ErrPOIListNotFound.Error())
}

func (s *POITestSuite) TestDeletePOI() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	var profile schema.Profile
	err := s.testDatabase.Collection(schema.ProfileCollection).FindOne(context.Background(), bson.M{
		"account_number": "account-test-delete-poi",
	}, options.FindOne().SetProjection(bson.M{"points_of_interest": 1})).Decode(&profile)
	s.NoError(err)
	s.Len(profile.PointsOfInterest, 2)

	s.NoError(store.DeletePOI("account-test-delete-poi", addedPOIID))

	err = s.testDatabase.Collection(schema.ProfileCollection).FindOne(context.Background(), bson.M{
		"account_number": "account-test-delete-poi",
	}, options.FindOne().SetProjection(bson.M{"points_of_interest": 1})).Decode(&profile)
	s.NoError(err)
	s.Len(profile.PointsOfInterest, 2)
	s.Equal(addedPOIID, profile.PointsOfInterest[0].ID)
	s.False(profile.PointsOfInterest[0].Monitored)
	s.Equal(addedPOIID2, profile.PointsOfInterest[1].ID)
	s.True(profile.PointsOfInterest[1].Monitored)

	s.NoError(store.DeletePOI("account-test-delete-poi", addedPOIID2))

	err = s.testDatabase.Collection(schema.ProfileCollection).FindOne(context.Background(), bson.M{
		"account_number": "account-test-delete-poi",
	}, options.FindOne().SetProjection(bson.M{"points_of_interest": 1})).Decode(&profile)
	s.NoError(err)
	s.Len(profile.PointsOfInterest, 2) // does not remove from profile
}

func (s *POITestSuite) TestDeletePOINonexistentPOI() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	s.NoError(store.DeletePOI("account-test-no-poi", addedPOIID))
}

func (s *POITestSuite) TestDeletePOIFromNonexistentAccount() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	s.NoError(store.DeletePOI("account-not-found-test-poi", addedPOIID))
}

func (s *POITestSuite) TestUpdatePOIAlias() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	var profile schema.Profile
	// before
	err := s.testDatabase.Collection(schema.ProfileCollection).FindOne(context.Background(), bson.M{
		"account_number": "account-test-update-poi-alias",
	}, options.FindOne().SetProjection(bson.M{"points_of_interest": 1})).Decode(&profile)
	s.NoError(err)
	s.Len(profile.PointsOfInterest, 1)
	s.Equal(originAlias, profile.PointsOfInterest[0].Alias)

	s.NoError(store.UpdatePOIAlias("account-test-update-poi-alias", "new-poi-alias", addedPOIID))

	// after
	err = s.testDatabase.Collection(schema.ProfileCollection).FindOne(context.Background(), bson.M{
		"account_number": "account-test-update-poi-alias",
	}, options.FindOne().SetProjection(bson.M{"points_of_interest": 1})).Decode(&profile)
	s.NoError(err)
	s.Equal("new-poi-alias", profile.PointsOfInterest[0].Alias)
}

func (s *POITestSuite) TestUpdatePOIAliasFromNonexistentAccount() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	s.EqualError(store.UpdatePOIAlias("account-not-found-test-poi", "new-poi-alias", addedPOIID), ErrPOINotFound.Error())
}

func (s *POITestSuite) TestUpdatePOIAliasForNotAddedPOI() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	s.EqualError(store.UpdatePOIAlias("account-test-update-poi-alias", "new-poi-alias", addedPOIID2), ErrPOINotFound.Error())
}

func (s *POITestSuite) TestUpdatePOIMetric() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	var poi schema.POI
	// before
	err := s.testDatabase.Collection(schema.POICollection).FindOne(context.Background(), bson.M{"_id": updatePOIID}).Decode(&poi)
	s.NoError(err)
	s.Equal(poi.Metric.Score, 0.0)
	s.Equal(poi.Score, 0.0)
	s.Equal(poi.ScoreDelta, 0.0)

	s.NoError(store.UpdatePOIMetric(updatePOIID, schema.Metric{Score: 55.66}, 94.87, 94.0))

	// after
	err = s.testDatabase.Collection(schema.POICollection).FindOne(context.Background(), bson.M{"_id": updatePOIID}).Decode(&poi)
	s.NoError(err)
	s.Equal(poi.Metric.Score, 55.66)
	s.Equal(poi.Score, 94.87)
	s.Equal(poi.ScoreDelta, 94.0)
}

func (s *POITestSuite) TestAddPOIResources() {
	var testResources = []schema.Resource{
		{Name: "test-new1"}, {Name: "test-new2"},
	}

	store := NewMongoStore(s.mongoClient, s.testDBName)
	resources, err := store.AddPOIResources(noResourcesPOIID, testResources, "en")
	s.NoError(err)
	s.Len(resources, 2)
}

func (s *POITestSuite) TestAddPOIResourcesWithIDButNoLanguageSupport() {
	delete(defaultResourceIDMap, "en")
	delete(defaultResourceList, "en")

	var testResources = []schema.Resource{
		{ID: "resource_1"}, {ID: "resource_2"},
	}

	store := NewMongoStore(s.mongoClient, s.testDBName)
	resources, err := store.AddPOIResources(noResourcesPOIID, testResources, "en")
	s.EqualError(err, "poi resources can not resolved")
	s.Nil(resources)
}

func (s *POITestSuite) TestAddPOIResourcesNotFound() {
	var testResources = []schema.Resource{
		{Name: "test-new1"}, {Name: "test-new2"},
	}

	store := NewMongoStore(s.mongoClient, s.testDBName)
	resources, err := store.AddPOIResources(notFoundPOIID, testResources, "en")
	s.Error(err)
	s.EqualError(err, "poi not found")
	s.Len(resources, 0)
}

func (s *POITestSuite) TestAddPOIResourcesDuplicated() {
	ctx := context.Background()

	store := NewMongoStore(s.mongoClient, s.testDBName)
	var duplicatedTestResources = []schema.Resource{
		{ID: "12c99e1e3afec0ce9c21d27794d21bfff496116b8ed9ba354f731d35cb0be6b5", Name: "test3"},
	}

	var poiBefore schema.POI
	err := s.testDatabase.Collection(schema.POICollection).
		FindOne(ctx,
			bson.M{"_id": duplicatedResourcesPOIID},
			options.FindOne().SetProjection(bson.M{"resource_ratings": 1})).Decode(&poiBefore)
	s.NoError(err)
	s.Equal(1, len(poiBefore.ResourceRatings.Resources))

	resources, err := store.AddPOIResources(duplicatedResourcesPOIID, duplicatedTestResources, "en")
	s.NoError(err)
	s.Len(resources, 1)

	var poiAfter schema.POI
	err = s.testDatabase.Collection(schema.POICollection).
		FindOne(ctx,
			bson.M{"_id": duplicatedResourcesPOIID},
			options.FindOne().SetProjection(bson.M{"resource_ratings": 1})).Decode(&poiAfter)
	s.NoError(err)
	s.Equal(1, len(poiAfter.ResourceRatings.Resources))
}

func (s *POITestSuite) TestGetPOIResourcesWithNoResourceAdded() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	resources, err := store.GetPOIResources(noResourcesPOIID2, false, false, "en")
	s.NoError(err)
	s.Len(resources, 126)

	resources, err = store.GetPOIResources(noResourcesPOIID2, true, false, "en")
	s.NoError(err)
	s.Len(resources, 30)
}

func (s *POITestSuite) TestGetPOIResourcesWithUnsupportLanguage() { // fallback to en
	store := NewMongoStore(s.mongoClient, s.testDBName)
	resources, err := store.GetPOIResources(noResourcesPOIID2, false, false, "fr")
	s.NoError(err)
	s.Len(resources, 126)

	resources, err = store.GetPOIResources(noResourcesPOIID2, true, false, "fr")
	s.NoError(err)
	s.Len(resources, 30)
}

func (s *POITestSuite) TestGetPOIResourcesWithNoDefaultLanguage() {
	delete(defaultResourceIDMap, "en")
	delete(defaultResourceList, "en")

	store := NewMongoStore(s.mongoClient, s.testDBName)
	resources, err := store.GetPOIResources(noResourcesPOIID2, false, false, "en")
	s.EqualError(err, "poi resources not found")
	s.Nil(resources)

	resources, err = store.GetPOIResources(noResourcesPOIID2, true, false, "en")
	s.EqualError(err, "poi resources not found")
	s.Nil(resources)
}

func (s *POITestSuite) TestGetPOIResourcesWithTwoResourceAdded() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	resources, err := store.GetPOIResources(officialResourcesPOIID, false, false, "en")
	s.NoError(err)
	s.Len(resources, 124) // two resources added, 126 - 2

	resources, err = store.GetPOIResources(officialResourcesPOIID, true, false, "en")
	s.NoError(err)
	s.Len(resources, 29) // two resources added but only one important, 30 - 1
}

func (s *POITestSuite) TestGetPOIResourcesWithTwoResourceAddedAndIncludeAdded() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	resources, err := store.GetPOIResources(officialResourcesPOIID, false, true, "en")
	s.NoError(err)
	s.Len(resources, 126) // two resources added, 126 - 2

	resources, err = store.GetPOIResources(officialResourcesPOIID, true, true, "en")
	s.NoError(err)
	s.Len(resources, 31) // two resources added but only one not belongs to important, 30 + 1
}
func (s *POITestSuite) TestGetPOIMetric() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	metric, err := store.GetPOIMetrics(metricPOIID)
	s.NoError(err)
	s.NotNil(metric)

	s.Equal(float64(87), metric.Score)
	s.IsType(int64(0), metric.LastUpdate)
}

func (s *POITestSuite) TestNearestPOIWithoutAnyPoint() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	location := schema.Location{
		Latitude:  0,
		Longitude: 0,
	}
	// search points from (0, 0) within 1km.
	poiIDs, err := store.NearestPOI(1, location)
	s.NoError(err)
	s.Nil(poiIDs)
}

func (s *POITestSuite) TestNearestPOIWithAPoint() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	location := schema.Location{
		Latitude:  25.12345,
		Longitude: 120.12345,
	}
	// search points from (25.12345, 120.12345) within 1km.
	poiIDs, err := store.NearestPOI(1, location)
	s.NoError(err)
	s.NotNil(poiIDs)
	s.Len(poiIDs, 1)
}

func (s *POITestSuite) TestGetResourceList() {
	list, err := getResourceList("", false)
	s.NoError(err)
	s.Len(list, DefaultResourceCount)
}

func (s *POITestSuite) TestGetResourceListImportant() {
	list, err := getResourceList("", true)
	s.NoError(err)
	s.Len(list, len(importantResourceID))
}

func (s *POITestSuite) TestListPOIByResourceNotAdded() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	location := schema.Location{
		Latitude:  25.12345,
		Longitude: 119.12345,
	}

	pois, err := store.ListPOIByResource("resource_not_added", location)
	s.NoError(err)
	s.Len(pois, 0)
}

func (s *POITestSuite) TestListPOIByResourceAddedButNoRating() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	location := schema.Location{
		Latitude:  25.12345,
		Longitude: 119.12345,
	}

	pois, err := store.ListPOIByResource("resource_no_rating", location)
	s.NoError(err)
	s.Len(pois, 0)
}

func (s *POITestSuite) TestListPOIByResourceNormal() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	location := schema.Location{
		Latitude:  25.12345,
		Longitude: 119.12345,
	}

	pois, err := store.ListPOIByResource("resource_11", location)
	s.NoError(err)
	s.Len(pois, 2)

	pois, err = store.ListPOIByResource("resource_12", location)
	s.NoError(err)
	s.Len(pois, 1)
}

func (s *POITestSuite) TestListPOIByResourceOutOfRange() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	location := schema.Location{
		Latitude:  20,
		Longitude: 120,
	}

	pois, err := store.ListPOIByResource("resource_11", location)
	s.NoError(err)
	s.Len(pois, 0)
}

func (s *POITestSuite) TestListPOIByPlaceType() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	pois, err := store.ListPOIByPlaceType("test_place")
	s.NoError(err)
	s.Len(pois, 2)

	pois, err = store.ListPOIByPlaceType("happy_run")
	s.NoError(err)
	s.Len(pois, 1)
}

func (s *POITestSuite) TestSearchPOIByText() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	pois, err := store.SearchPOIByText("bitmark")
	s.NoError(err)
	s.Len(pois, 2)

	pois, err = store.SearchPOIByText("Bitmark")
	s.NoError(err)
	s.Len(pois, 2)

	pois, err = store.SearchPOIByText("address123")
	s.NoError(err)
	s.Len(pois, 1)
	s.Contains(pois[0].Address, "address123")

	pois, err = store.SearchPOIByText("alias123")
	s.NoError(err)
	s.Len(pois, 1)
	s.Contains(pois[0].Alias, "alias123")
}

func (s *POITestSuite) TestGetPOIByCoordinates() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	location := schema.Location{
		Latitude:  25.123,
		Longitude: 120.123,
	}

	poi, err := store.GetPOIByCoordinates(location)
	s.NoError(err)
	s.NotNil(poi)
	s.Equal("Taiwan", poi.Country)
	s.Equal("", poi.State)
	s.Equal("Yilan County", poi.County)
	s.Equal([]float64{location.Longitude, location.Latitude}, poi.Location.Coordinates)
}

func (s *POITestSuite) TestGetPOIByCoordinatesNoPOI() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	location := schema.Location{
		Latitude:  25.1111,
		Longitude: 120.1111,
	}

	poi, err := store.GetPOIByCoordinates(location)
	s.EqualError(err, ErrPOINotFound.Error())
	s.Nil(poi)
}

func (s *POITestSuite) TestListAllPOI() {
	store := NewMongoStore(s.mongoClient, testListAllDBName)

	poi, err := store.ListAllPOI(0, 0)
	s.NoError(err)
	s.Len(poi, 2)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to s.Run
func TestPOITestSuite(t *testing.T) {
	suite.Run(t, NewPOITestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
