package geo

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"googlemaps.github.io/maps"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/share/geojson"
)

type ResolverTestSuite struct {
	suite.Suite
	reloadDB     bool
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

var USLocationTestData = []schema.Location{
	{Latitude: 38.876408, Longitude: -77.433901, AddressComponent: schema.AddressComponent{Country: "United States", State: "Virginia", County: "Fairfax County"}},
	{Latitude: 33.068212, Longitude: -116.765943, AddressComponent: schema.AddressComponent{Country: "United States", State: "California", County: "San Diego County"}},
	{Latitude: 30.151855, Longitude: -84.522030, AddressComponent: schema.AddressComponent{Country: "United States", State: "Florida", County: "Wakulla County"}},
	{Latitude: 40.776032, Longitude: -73.959463, AddressComponent: schema.AddressComponent{Country: "United States", State: "New York", County: "New York County"}},
	{Latitude: 69.553596, Longitude: -157.072916, AddressComponent: schema.AddressComponent{Country: "United States", State: "Alaska", County: "North Slope Borough"}},
	{Latitude: 32.601252, Longitude: -92.671741, AddressComponent: schema.AddressComponent{Country: "United States", State: "Louisiana", County: "Lincoln Parish"}},
}

var OtherLocationTestData = []schema.Location{
	{Latitude: 64.1499893, Longitude: -21.954031, AddressComponent: schema.AddressComponent{Country: "Iceland", State: "", County: ""}},
	{Latitude: 16.056142, Longitude: 108.200845, AddressComponent: schema.AddressComponent{Country: "Vietnam", State: "", County: ""}},
	{Latitude: 16.043864, Longitude: 108.217694, AddressComponent: schema.AddressComponent{Country: "Vietnam", State: "", County: ""}},
	{Latitude: 35.6599743, Longitude: 139.7432433, AddressComponent: schema.AddressComponent{Country: "Japan", State: "", County: ""}},
	{Latitude: 49.0096941, Longitude: 2.5457358, AddressComponent: schema.AddressComponent{Country: "France", State: "", County: ""}},
}

func NewResolverTestSuite(connURI, dbName, mapAPIKey string, reloadDB bool) *ResolverTestSuite {
	return &ResolverTestSuite{
		reloadDB:   reloadDB,
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

	if s.mapAPIKey != "" {
		mapClient, err := maps.NewClient(maps.WithAPIKey(s.mapAPIKey))
		if err != nil {
			s.T().Fatalf("init goolge map client with error: %s", err.Error())
		}
		s.mapClient = mapClient
	}

	s.mongoClient = mongoClient
	s.testDatabase = mongoClient.Database(s.testDBName)

	n, err := s.testDatabase.Collection("boundary").CountDocuments(context.Background(), bson.M{})
	if err != nil {
		s.T().Fatal(err)
	}

	// The boundary data is large. Therefore, we only load the data if
	// the `RELOAD_DB` is set or there is no items in the collection.
	if s.reloadDB || n == 0 {
		if err := s.CleanMongoDB(); err != nil {
			s.T().Fatal(err)
		}
		schema.NewMongoDBIndexer(s.connURI, s.testDBName).IndexAll()
		if err := s.LoadMongoDBFixtures(); err != nil {
			s.T().Fatal(err)
		}
	}
}

func (s *ResolverTestSuite) LoadMongoDBFixtures() error {
	if err := geojson.ImportTaiwanBoundary(s.mongoClient, s.testDBName, "../share/geojson/tw-boundary.json"); err != nil {
		return err
	}

	if err := geojson.ImportUSBoundary(s.mongoClient, s.testDBName, "../share/geojson/us-boundary.geojson"); err != nil {
		return err
	}

	return geojson.ImportWorldCountryBoundary(s.mongoClient, s.testDBName, "../share/geojson/world-boundary.geojson")
}

func (s *ResolverTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

func (s *ResolverTestSuite) TestGeocodingLocationResolverForTaiwan() {
	if s.mapAPIKey == "" {
		s.T().Skip("Skip resolver tests due to missing map api key")
	}

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

func (s *ResolverTestSuite) TestGeocodingLocationResolverForUSLocation() {
	if s.mapAPIKey == "" {
		s.T().Skip("Skip resolver tests due to missing map api key")
	}

	r := NewGeocodingLocationResolver(s.mapClient)

	for _, testdata := range USLocationTestData {
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
	if s.mapAPIKey == "" {
		s.T().Skip("Skip resolver tests due to missing map api key")
	}

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

func (s *ResolverTestSuite) TestMongodbLocationResolverForUSLocation() {
	r := NewMongodbLocationResolver(s.mongoClient, s.testDBName)

	for _, testdata := range USLocationTestData {
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

func (s *ResolverTestSuite) TestMongodbLocationResolverForOtherLocation() {
	r := NewMongodbLocationResolver(s.mongoClient, s.testDBName)

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
	if s.mapAPIKey == "" {
		s.T().Skip("Skip resolver tests due to missing map api key")
	}

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

func (s *ResolverTestSuite) TestMultipleLocationResolverForUSLocation() {
	if s.mapAPIKey == "" {
		s.T().Skip("Skip resolver tests due to missing map api key")
	}

	r := NewMultipleLocationResolver(
		NewMongodbLocationResolver(s.mongoClient, s.testDBName),
		NewGeocodingLocationResolver(s.mapClient),
	)

	for _, testdata := range USLocationTestData {
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
	if s.mapAPIKey == "" {
		s.T().Skip("Skip resolver tests due to missing map api key")
	}

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
	if s.mapAPIKey == "" {
		s.T().Skip("Skip resolver tests due to missing map api key")
	}

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
	reloadDB := os.Getenv("RELOAD_DB") != ""
	suite.Run(t, NewResolverTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-geo", mapKey, reloadDB))
}
