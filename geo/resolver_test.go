package geo

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"googlemaps.github.io/maps"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type ResolverTestSuite struct {
	suite.Suite
	connURI      string
	testDBName   string
	mapAPIKey    string
	mapClient    *maps.Client
	mongoClient  *mongo.Client
	testDatabase *mongo.Database
}

var TaiwanLocationTestData = []schema.Location{
	{Latitude: 25.1317044, Longitude: 121.7380282, AddressComponent: schema.AddressComponent{Country: "Taiwan", State: "", County: "Keelung City"}},
	{Latitude: 24.6828385, Longitude: 121.7772541, AddressComponent: schema.AddressComponent{Country: "Taiwan", State: "", County: "Yilan County"}},
	{Latitude: 23.095489, Longitude: 121.360916, AddressComponent: schema.AddressComponent{Country: "Taiwan", State: "", County: "Taitung County"}},
	{Latitude: 24.796029, Longitude: 120.994767, AddressComponent: schema.AddressComponent{Country: "Taiwan", State: "", County: "Hsinchu City"}},
	{Latitude: 25.147192, Longitude: 121.596010, AddressComponent: schema.AddressComponent{Country: "Taiwan", State: "", County: "New Taipei City"}},
	{Latitude: 25.147057, Longitude: 121.593191, AddressComponent: schema.AddressComponent{Country: "Taiwan", State: "", County: "Taipei City"}},
}

var OtherLocationTestData = []schema.Location{
	{Latitude: 33.036147, Longitude: -117.2842319, AddressComponent: schema.AddressComponent{Country: "United States", State: "California", County: "San Diego County"}},
	{Latitude: 40.733992, Longitude: -73.993641, AddressComponent: schema.AddressComponent{Country: "United States", State: "New York", County: "New York County"}},
	{Latitude: 64.1499893, Longitude: -21.954031, AddressComponent: schema.AddressComponent{Country: "Iceland", State: "", County: "Capital Region"}},
}

func NewResolverTestSuite(connURI, dbName, mapAPIKey string) *ResolverTestSuite {
	return &ResolverTestSuite{
		connURI:    connURI,
		testDBName: dbName,
		mapAPIKey:  mapAPIKey,
	}
}

func (s *ResolverTestSuite) SetupSuite() {
	if s.connURI == "" || s.testDBName == "" {
		s.T().Fatal("invalid test suite configuration")
	}

	opts := options.Client().ApplyURI(s.connURI)
	mongoClient, err := mongo.NewClient(opts)
	if nil != err {
		s.T().Fatalf("create mongo client with error: %s", err)
	}

	if err := mongoClient.Connect(context.Background()); nil != err {
		s.T().Fatalf("connect mongo database with error: %s", err.Error())
	}

	mapClient, err := maps.NewClient(maps.WithAPIKey(s.mapAPIKey))
	if err != nil {
		s.T().Fatalf("init goolge map client with error: %s", err.Error())
	}

	s.mapClient = mapClient
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

func (s *ResolverTestSuite) LoadMongoDBFixtures() error {
	type GeoFeature struct {
		Type       string            `json:"type"`
		Properties map[string]string `json:"properties"`
		Geometry   schema.Geometry   `json:"geometry"`
	}

	type GeoJSONTW struct {
		Name     string       `json:"name"`
		Features []GeoFeature `json:"features"`
	}

	var result GeoJSONTW

	file, err := os.Open("../share/geojson/tw-boundary.json")
	if err != nil {
		return err
	}

	if err := json.NewDecoder(file).Decode(&result); err != nil {
		return err
	}
	var boundaries []interface{}
	for _, b := range result.Features {
		boundaries = append(boundaries, schema.Boundary{
			Country:  "Taiwan",
			State:    "",
			County:   b.Properties["COUNTYENG"],
			Geometry: b.Geometry,
		})
	}

	if _, err := s.testDatabase.Collection(schema.BoundaryCollection).InsertMany(context.Background(), boundaries); err != nil {
		return err
	}

	return nil
}

func (s *ResolverTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

func (s *ResolverTestSuite) TestGeocodingLocationResolverForTaiwan() {
	r := NewGeocodingLocationResolver(s.mapClient)

	for _, testdata := range TaiwanLocationTestData {
		location, err := r.GetPoliticalInfo(schema.Location{
			Latitude:  testdata.Latitude,
			Longitude: testdata.Longitude,
		})

		s.NoError(err)
		s.Equal(testdata.Country, location.Country)
		s.Equal(testdata.State, location.State)
		s.Equal(testdata.County, location.County)
	}
}

func (s *ResolverTestSuite) TestGeocodingLocationResolverForOtherLocation() {
	r := NewGeocodingLocationResolver(s.mapClient)

	for _, testdata := range OtherLocationTestData {
		location, err := r.GetPoliticalInfo(schema.Location{
			Latitude:  testdata.Latitude,
			Longitude: testdata.Longitude,
		})

		s.NoError(err)
		s.Equal(testdata.Country, location.Country)
		s.Equal(testdata.State, location.State)
		s.Equal(testdata.County, location.County)
	}
}

func (s *ResolverTestSuite) TestGeocodingLocationResolverNotFound() {
	r := NewGeocodingLocationResolver(s.mapClient)

	location, err := r.GetPoliticalInfo(schema.Location{ // sea near by Hsinchu
		Latitude:  24.9338699,
		Longitude: 120.9536467,
	})

	s.Error(err)
	s.EqualError(err, "no geo information found")
	s.Equal("", location.Country)
	s.Equal("", location.State)
	s.Equal("", location.County)
}

func (s *ResolverTestSuite) TestMongodbLocationResolverForTaiwan() {
	r := NewMongodbLocationResolver(s.mongoClient, s.testDBName)

	for _, testdata := range TaiwanLocationTestData {
		location, err := r.GetPoliticalInfo(schema.Location{
			Latitude:  testdata.Latitude,
			Longitude: testdata.Longitude,
		})

		s.NoError(err)
		s.Equal(testdata.Country, location.Country)
		s.Equal(testdata.State, location.State)
		s.Equal(testdata.County, location.County)
	}
}

func (s *ResolverTestSuite) TestMongodbLocationResolverNotFound() {
	r := NewMongodbLocationResolver(s.mongoClient, s.testDBName)

	location, err := r.GetPoliticalInfo(schema.Location{ // sea near by Hsinchu
		Latitude:  24.9338699,
		Longitude: 120.9536467,
	})

	s.Error(err)
	s.EqualError(err, "no geo information found")
	s.Equal("", location.Country)
	s.Equal("", location.State)
	s.Equal("", location.County)
}

func (s *ResolverTestSuite) TestMultipleLocationResolverForTaiwan() {
	r := NewMultipleLocationResolver(
		NewMongodbLocationResolver(s.mongoClient, s.testDBName),
		NewGeocodingLocationResolver(s.mapClient),
	)

	for _, testdata := range TaiwanLocationTestData {
		location, err := r.GetPoliticalInfo(schema.Location{
			Latitude:  testdata.Latitude,
			Longitude: testdata.Longitude,
		})

		s.NoError(err)
		s.Equal(testdata.Country, location.Country)
		s.Equal(testdata.State, location.State)
		s.Equal(testdata.County, location.County)
	}
}

func (s *ResolverTestSuite) TestMultipleLocationResolverForOtherLocation() {
	r := NewMultipleLocationResolver(
		NewMongodbLocationResolver(s.mongoClient, s.testDBName),
		NewGeocodingLocationResolver(s.mapClient),
	)

	for _, testdata := range OtherLocationTestData {
		location, err := r.GetPoliticalInfo(schema.Location{
			Latitude:  testdata.Latitude,
			Longitude: testdata.Longitude,
		})

		s.NoError(err)
		s.Equal(testdata.Country, location.Country)
		s.Equal(testdata.State, location.State)
		s.Equal(testdata.County, location.County)
	}
}

func (s *ResolverTestSuite) TestMultipleLocationResolverMongodbNotFound() {
	r := NewMultipleLocationResolver(
		NewMongodbLocationResolver(s.mongoClient, s.testDBName),
		NewGeocodingLocationResolver(s.mapClient),
	)

	location, err := r.GetPoliticalInfo(schema.Location{ // sea near by Hsinchu
		Latitude:  24.9338699,
		Longitude: 120.9536467,
	})

	s.Error(err)
	s.EqualError(err, "#0: no geo information found\n#1: no geo information found")
	e, ok := err.(*MultipleResolverErrors)
	s.True(ok)
	s.Len(e.errors, 2)
	s.Equal("", location.Country)
	s.Equal("", location.State)
	s.Equal("", location.County)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to s.Run
func TestResolverTestSuite(t *testing.T) {
	mapKey := os.Getenv("MAP_APIKEY")
	if mapKey == "" {
		t.Skip("Skip resolver tests due to missing map api key")
	}
	suite.Run(t, NewResolverTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db", mapKey))
}
