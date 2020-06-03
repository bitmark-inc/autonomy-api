package store

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"googlemaps.github.io/maps"

	"github.com/bitmark-inc/autonomy-api/geo"
	"github.com/bitmark-inc/autonomy-api/geo/mocks"
	"github.com/bitmark-inc/autonomy-api/schema"
)

var addedPOIID = primitive.NewObjectID()
var addedPOIID2 = primitive.NewObjectID()
var existedPOIID = primitive.NewObjectID()
var noCountryPOIID = primitive.NewObjectID()
var metricPOIID = primitive.NewObjectID()

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
	}

	addedPOI = schema.POI{
		ID: addedPOIID,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{120.123, 25.123},
		},
		Country: "Taiwan",
		State:   "",
		County:  "Yilan County",
	}

	addedPOI2 = schema.POI{
		ID: addedPOIID2,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{120.1234, 25.1234},
		},
		Country: "Taiwan",
		State:   "",
		County:  "Taipei City",
	}

	existedPOI = schema.POI{
		ID: existedPOIID,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{120.12, 25.12},
		},
		Country: "Taiwan",
		State:   "",
		County:  "Yilan County",
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
	}
)

var originAlias = "origin POI"

type POITestSuite struct {
	suite.Suite
	connURI      string
	testDBName   string
	mongoClient  *mongo.Client
	testDatabase *mongo.Database
	mockResolver *mocks.MockLocationResolver
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
					ID:    addedPOIID,
					Alias: originAlias,
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
		addedPOI,
		addedPOI2,
		existedPOI,
		metricPOI,
	}); err != nil {
		return err
	}

	return nil
}

// CleanMongoDB drop the whole test mongodb
func (s *POITestSuite) CleanMongoDB() error {
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

// TestAddPOIWithNonExistAccount tests adding a new poi by an account which is not existent
func (s *POITestSuite) TestAddPOIWithNonExistAccount() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	s.mockResolver.EXPECT().
		GetPoliticalInfo(gomock.AssignableToTypeOf(schema.Location{})).
		Return(testLocation, nil)

	poi, err := store.AddPOI("account-not-found-test-poi", "test-poi", "", 120, 25)
	s.EqualError(err, "fail to update poi into profile")
	s.Nil(poi)
}

// TestAddPOI tests adding a new poi normally
func (s *POITestSuite) TestAddPOI() {
	ctx := context.Background()
	store := NewMongoStore(s.mongoClient, s.testDBName)

	s.mockResolver.EXPECT().
		GetPoliticalInfo(gomock.AssignableToTypeOf(schema.Location{})).
		Return(testLocation, nil)

	poi, err := store.AddPOI("account-test-add-poi", "test-poi", "", 120.1, 25.1)
	s.NoError(err)
	s.Equal("United States", poi.Country)
	s.Equal("New York", poi.State)
	s.Equal("New York County", poi.County)
	s.Equal([]float64{120.1, 25.1}, poi.Location.Coordinates)

	count, err := s.testDatabase.Collection(schema.POICollection).CountDocuments(ctx, bson.M{"_id": poi.ID})
	s.NoError(err)
	s.Equal(int64(1), count)

	count, err = s.testDatabase.Collection(schema.ProfileCollection).CountDocuments(context.Background(), bson.M{
		"account_number":           "account-test-add-poi",
		"points_of_interest.id":    poi.ID,
		"points_of_interest.alias": "test-poi",
	})
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

	count, err := s.testDatabase.Collection(schema.ProfileCollection).CountDocuments(context.Background(), bson.M{
		"account_number":        "account-test-add-poi",
		"points_of_interest.id": existedPOIID,
	})
	s.NoError(err)
	s.Equal(int64(0), count)

	poi, err := store.AddPOI("account-test-add-poi", "test-existent-poi", "", existedPOI.Location.Coordinates[0], existedPOI.Location.Coordinates[1])
	s.NoError(err)
	s.Equal("Taiwan", poi.Country)
	s.Equal("", poi.State)
	s.Equal("Yilan County", poi.County)
	s.Equal([]float64{existedPOI.Location.Coordinates[0], existedPOI.Location.Coordinates[1]}, poi.Location.Coordinates)

	count, err = s.testDatabase.Collection(schema.POICollection).CountDocuments(ctx, bson.M{"_id": existedPOIID})
	s.NoError(err)
	s.Equal(int64(1), count)

	count, err = s.testDatabase.Collection(schema.ProfileCollection).CountDocuments(context.Background(), bson.M{
		"account_number":           "account-test-add-poi",
		"points_of_interest.id":    existedPOIID,
		"points_of_interest.alias": "test-existent-poi",
	})
	s.NoError(err)
	s.Equal(int64(1), count)
}

// TestAddDuplicatedPOI tests adding a duplicated poi where its coordinates has alreday existed
// for the test account
func (s *POITestSuite) TestAddDuplicatedPOI() {
	ctx := context.Background()
	store := NewMongoStore(s.mongoClient, s.testDBName)

	s.mockResolver.EXPECT().
		GetPoliticalInfo(gomock.AssignableToTypeOf(schema.Location{})).
		Return(testLocation, nil)

		// poi is not in the profile at beginning
	count, err := s.testDatabase.Collection(schema.ProfileCollection).CountDocuments(context.Background(), bson.M{
		"account_number":        "account-test-add-poi",
		"points_of_interest.id": addedPOIID,
	})
	s.NoError(err)
	s.Equal(int64(1), count)

	// use a different name to add an added poi
	poi, err := store.AddPOI("account-test-add-poi", "test-duplicated-add-poi", "", addedPOI.Location.Coordinates[0], addedPOI.Location.Coordinates[1])
	s.NoError(err)
	s.Equal("Taiwan", poi.Country)
	s.Equal("", poi.State)
	s.Equal("Yilan County", poi.County)
	s.Equal([]float64{addedPOI.Location.Coordinates[0], addedPOI.Location.Coordinates[1]}, poi.Location.Coordinates)

	count, err = s.testDatabase.Collection(schema.POICollection).CountDocuments(ctx, bson.M{"_id": addedPOIID})
	s.NoError(err)
	s.Equal(int64(1), count)

	// the alias of the added poi will not updated
	count, err = s.testDatabase.Collection(schema.ProfileCollection).CountDocuments(context.Background(), bson.M{
		"account_number":           "account-test-add-poi",
		"points_of_interest.id":    addedPOIID,
		"points_of_interest.alias": originAlias,
	})
	s.NoError(err)
	s.Equal(int64(1), count)
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

func (s *POITestSuite) TestUpdatePOIOrderMismatch() {
	store := NewMongoStore(s.mongoClient, s.testDBName)

	err := store.UpdatePOIOrder("account-test-poi-reorder", []string{addedPOIID.Hex()})
	s.EqualError(err, ErrPOIListMismatch.Error())
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
	s.Len(profile.PointsOfInterest, 1)
	s.Equal(addedPOIID2, profile.PointsOfInterest[0].ID)

	s.NoError(store.DeletePOI("account-test-delete-poi", addedPOIID2))

	err = s.testDatabase.Collection(schema.ProfileCollection).FindOne(context.Background(), bson.M{
		"account_number": "account-test-delete-poi",
	}, options.FindOne().SetProjection(bson.M{"points_of_interest": 1})).Decode(&profile)
	s.NoError(err)
	s.Len(profile.PointsOfInterest, 0)
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

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to s.Run
func TestPOITestSuite(t *testing.T) {
	suite.Run(t, NewPOITestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
