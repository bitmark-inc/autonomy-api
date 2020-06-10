package store

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/consts"
	"github.com/bitmark-inc/autonomy-api/schema"
)

const (
	numberOfConfirmTaiwan = 29
)

type ConfirmCDSTestSuite struct {
	suite.Suite
	connURI      string
	testDBName   string
	mongoClient  *mongo.Client
	testDatabase *mongo.Database
	ConfirmExpected
}

type ConfirmExpected struct {
	ExpectContinue           []schema.CDSScoreDataSet
	ExpectContinueTimeBefore []schema.CDSScoreDataSet
	ExpectActiveConfirm
	ExpectActiveNoDataSet ExpectActiveConfirm
}
type ExpectActiveConfirm struct {
	Active              float64
	Delta               float64
	RateChangeRoundEven float64
}

func NewConfirmTestSuite(connURI, dbName string) *ConfirmCDSTestSuite {
	return &ConfirmCDSTestSuite{
		connURI:    connURI,
		testDBName: dbName,
	}
}

func (s *ConfirmCDSTestSuite) SetupSuite() {
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
	schema.NewMongoDBIndexer(s.connURI, s.testDBName).IndexCDSConfirmCollection()
	s.LoadExpectedData()
}

func (s *ConfirmCDSTestSuite) SetupTest() {
	err := s.RemoveAllDocument()
	if err != nil {
		s.T().Fatal(err)
	}
	err = s.LoadMongoDBFixtures()
	if err != nil {
		s.T().Fatal(err)
	}
	s.ExpectDocCount(schema.CdsTaiwan, numberOfConfirmTaiwan)
}

// CleanMongoDB drop the whole test mongodb
func (s *ConfirmCDSTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

// LoadMongoDBFixtures will preload fixtures into test mongodb
func (s *ConfirmCDSTestSuite) LoadMongoDBFixtures() error {
	f := "fixtures/confirm_taiwan.json"
	data, err := s.LoadConfirmFixtures(f)
	if err != nil {
		return err
	}
	if len(data) != numberOfConfirmTaiwan {
		return fmt.Errorf("not a expected number of records")
	}
	if data[0].Name != schema.CdsTaiwan {
		return fmt.Errorf("not a expected data set")
	}
	collection, ok := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(schema.CdsTaiwan)]
	if !ok {
		s.T().Fatal("not a supported collection")
	}
	for i := 0; i < len(data); i++ {
		s.testDatabase.Collection(collection).InsertOne(context.Background(), data[i])
	}
	return nil
}

func (s *ConfirmCDSTestSuite) LoadConfirmFixtures(file string) ([]schema.CDSData, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var testData []schema.CDSData
	if err := json.NewDecoder(f).Decode(&testData); err != nil {
		return nil, err
	}
	return testData, nil
}

func (s *ConfirmCDSTestSuite) RemoveAllDocument() error {
	collection, ok := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(schema.CdsTaiwan)]
	if !ok {
		s.T().Fatal("not a supported collection")
	}
	_, err := s.testDatabase.Collection(collection).DeleteMany(context.Background(), bson.M{})
	if err != nil {
		return err
	}
	return nil
}

func (s *ConfirmCDSTestSuite) ExpectDocCount(country string, expectCount int64) {
	collection, ok := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(schema.CdsTaiwan)]
	s.True(ok)
	count, err := s.testDatabase.Collection(collection).CountDocuments(context.Background(), bson.M{})
	s.NoError(err)
	s.Equal(expectCount, count)
}

func (s *ConfirmCDSTestSuite) LoadExpectedData() {
	s.ExpectContinue = []schema.CDSScoreDataSet{
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 1},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
	}
	s.ExpectContinueTimeBefore = []schema.CDSScoreDataSet{
		{Cases: 1},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
		{Cases: 0},
	}

	s.ExpectActiveConfirm = ExpectActiveConfirm{Active: 19, Delta: -1, RateChangeRoundEven: -5}
	s.ExpectActiveNoDataSet = ExpectActiveConfirm{Active: 0, Delta: 0, RateChangeRoundEven: 0}
}

func (s *ConfirmCDSTestSuite) TestCreateCDS() {
	err := s.RemoveAllDocument()
	s.NoError(err)
	f := "fixtures/confirm_taiwan.json"
	data, err := s.LoadConfirmFixtures(f)
	s.NoError(err)
	s.Equal(numberOfConfirmTaiwan, len(data))
	store := NewMongoStore(s.mongoClient, s.testDBName)
	err = store.CreateCDS(data, schema.CdsTaiwan)
	s.NoError(err)
	// Test Duplicate
	err = store.CreateCDS(data, schema.CdsTaiwan)
	s.NoError(err)
	s.ExpectDocCount(schema.CdsTaiwan, numberOfConfirmTaiwan)
}

func (s *ConfirmCDSTestSuite) TestReplaceCDS() {
	s.ExpectDocCount(schema.CdsTaiwan, numberOfConfirmTaiwan)
	collection, ok := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(schema.CdsTaiwan)]
	s.True(ok)
	opts := options.Find().SetLimit(2)
	filter := bson.M{}
	ctx := context.Background()
	cur, err := s.testDatabase.Collection(collection).Find(ctx, filter, opts)
	s.NoError(err)
	var results []schema.CDSData
	for cur.Next(ctx) {
		var result schema.CDSData
		err = cur.Decode(&result)
		s.NoError(err)
		results = append(results, result)
	}
	s.Equal(2, len(results))
	replaceCases := []float64{100, 200}
	originalCases := []float64{0, 0}
	store := NewMongoStore(s.mongoClient, s.testDBName)
	for i := 0; i < len(results); i++ {
		originalCases[i] = results[i].Cases
		results[i].Cases = replaceCases[i]
	}
	err = store.ReplaceCDS(results, schema.CdsTaiwan)
	for i := 0; i < len(results); i++ {
		filter = bson.M{"name": results[i].Name, "report_ts": results[i].ReportTime}
		cur, err = s.testDatabase.Collection(collection).Find(ctx, filter)
		s.True(cur.Next(ctx))
		var queryReturn schema.CDSData
		cur.Decode(&queryReturn)
		s.Equal(float64(replaceCases[i]), queryReturn.Cases)
		s.False(cur.Next(ctx))
		cur.Close(ctx)
		queryReturn.Cases = originalCases[i]
		store.ReplaceCDS([]schema.CDSData{queryReturn}, schema.CdsTaiwan)
	}
	s.ExpectDocCount(schema.CdsTaiwan, numberOfConfirmTaiwan)
}

func (s *ConfirmCDSTestSuite) TestGetCDSActive() {
	// [2020-05-26] active = 19.0
	// [2020-05-25] N/A
	// [2020-05-24] active = 20.0
	loc := schema.Location{AddressComponent: schema.AddressComponent{Country: schema.CdsTaiwan}}
	referenceTime := time.Date(2020, 5, 26, 8, 0, 0, 0, time.UTC).UTC().Unix()
	store := NewMongoStore(s.mongoClient, s.testDBName)
	active, delta, changeRate, err := store.GetCDSActive(loc, referenceTime)
	s.NoError(err)
	s.Equal(s.ConfirmExpected.ExpectActiveConfirm.Active, active)
	s.Equal(s.ConfirmExpected.ExpectActiveConfirm.Delta, delta)
	s.Equal(s.ConfirmExpected.ExpectActiveConfirm.RateChangeRoundEven, math.RoundToEven(changeRate))

	loc = schema.Location{AddressComponent: schema.AddressComponent{Country: "Neverland"}}
	active, delta, changeRate, err = store.GetCDSActive(loc, referenceTime)
	s.Equal(err, ErrNoConfirmDataset)
	s.Equal(s.ConfirmExpected.ExpectActiveNoDataSet.Active, active)
	s.Equal(s.ConfirmExpected.ExpectActiveNoDataSet.Delta, delta)
	s.Equal(s.ConfirmExpected.ExpectActiveNoDataSet.RateChangeRoundEven, math.RoundToEven(changeRate))
}

func (s *ConfirmCDSTestSuite) TestContinuousDataCDSConfirm() {
	loc := schema.Location{AddressComponent: schema.AddressComponent{Country: schema.CdsTaiwan}}
	store := NewMongoStore(s.mongoClient, s.testDBName)
	dataset, err := store.ContinuousDataCDSConfirm(loc, consts.ConfirmScoreWindowSize, 0)
	s.NoError(err)
	s.Equal(14, len(dataset))
	for i := 0; i < len(dataset); i++ {
		s.Equal(float64(s.ConfirmExpected.ExpectContinue[i].Cases), dataset[i].Cases)
	}
	dataset, err = store.ContinuousDataCDSConfirm(loc, consts.ConfirmScoreWindowSize, 1589904000)
	s.NoError(err)
	s.Equal(14, len(dataset))
	for i := 0; i < len(dataset); i++ {
		s.Equal(float64(s.ConfirmExpected.ExpectContinueTimeBefore[i].Cases), dataset[i].Cases)
	}
}

func (s *ConfirmCDSTestSuite) TestDeleteCDSUnused() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	err := store.DeleteCDSUnused(schema.CdsTaiwan, 1589126400)
	s.NoError(err)
	s.ExpectDocCount(schema.CdsTaiwan, 14)
	err = store.DeleteCDSUnused(schema.CdsTaiwan, 2589126400)
	s.NoError(err)
	s.ExpectDocCount(schema.CdsTaiwan, 0)
}

func TestConfirmTestSuite(t *testing.T) {
	suite.Run(t, NewConfirmTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
