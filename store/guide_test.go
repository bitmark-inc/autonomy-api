package store

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type GuideTestSuite struct {
	suite.Suite
	connURI                     string
	testDBName                  string
	mongoClient                 *mongo.Client
	testDatabase                *mongo.Database
	testCenterFile              string
	ExpectedRecordCount         int64
	Centers                     []schema.TestCenter
	ExpectedCountReturn         int64
	LocationSet                 []schema.Location
	ExpectNearestInsitutionCode []string
}

func NewGuideTestSuite(connURI, dbName string) *GuideTestSuite {
	return &GuideTestSuite{
		connURI:    connURI,
		testDBName: dbName,
	}
}

func (s *GuideTestSuite) CleanMongoDB() error {
	return s.testDatabase.Drop(context.Background())
}

func (s *GuideTestSuite) SetupSuite() {
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

	schema.NewMongoDBIndexer(s.connURI, s.testDBName).IndexGuideCollection()
	s.ExpectedRecordCount = 162
	s.ExpectedCountReturn = 5
	s.testCenterFile = "./fixtures/TaiwanCDCTestCenter.csv"
	centers, err := loadTestCenter(s.testCenterFile)
	if err != nil {
		s.T().Fatal(err)
	}
	s.Centers = centers
	s.CreateTestCenterDB()
	s.LocationSet = append(s.LocationSet, schema.Location{
		Latitude:  float64(25.06485243),
		Longitude: float64(121.611873),
	})
	s.ExpectNearestInsitutionCode = []string{"1101110026", "101090517", "1131110516", "501010019", "1501010010"}
}

func (s *GuideTestSuite) TestNearbyTestCenter() {
	store := NewMongoStore(s.mongoClient, s.testDBName)
	centers, err := store.NearbyTestCenter(s.LocationSet[0], s.ExpectedCountReturn)
	s.NoError(err)
	s.Equal(s.ExpectedCountReturn, int64(len(centers)))
	for idx, center := range centers {
		s.Equal(s.ExpectNearestInsitutionCode[idx], center.InstitutionCode)
	}
}

func (s *GuideTestSuite) ExpectDocCount(expectCount int64) {
	count, err := s.testDatabase.Collection(schema.TestCenterCollection).CountDocuments(context.Background(), bson.M{})
	s.NoError(err)
	s.Equal(expectCount, count)
}

func (s *GuideTestSuite) CreateTestCenterDB() {
	centersToInterface := make([]interface{}, 0, len(s.Centers))
	for _, c := range s.Centers {
		centersToInterface = append(centersToInterface, c)
	}
	_, err := s.testDatabase.Collection(schema.TestCenterCollection).InsertMany(context.Background(), centersToInterface)
	if err != nil {
		s.T().Fatal()
	}
	s.ExpectDocCount(int64(len(s.Centers)))
}

func loadTestCenter(filepath string) ([]schema.TestCenter, error) {
	testingCenter, err := os.Open(filepath)
	if err != nil {
		return []schema.TestCenter{}, err
	}
	centers := []schema.TestCenter{}
	r := csv.NewReader(testingCenter)
	for {
		// Read each record from csv
		record, err := r.Read()

		if err != nil {
			if err == io.EOF {
				break
			}
			return centers, err
		}
		switch record[0] {
		case schema.CdsTaiwan:
			if len(record) < 8 {
				continue
			}
			lat, err := strconv.ParseFloat(record[4], 64)
			if err != nil {
				continue
			}
			long, err := strconv.ParseFloat(record[5], 64)
			if err != nil {
				continue
			}
			center := schema.TestCenter{
				Country:         schema.CDSCountryType(record[0]),
				County:          record[1],
				InstitutionCode: record[2],
				Location:        schema.GeoJSON{Type: "Point", Coordinates: []float64{long, lat}},
				Name:            record[3],
				Address:         record[6],
				Phone:           record[7],
			}
			centers = append(centers, center)
		}
	}
	return centers, nil
}

func (s *GuideTestSuite) DeleteAllDocument() {
	res, err := s.testDatabase.Collection(schema.TestCenterCollection).DeleteMany(context.Background(), bson.M{})
	s.NoError(err)
	fmt.Println("delete count", res.DeletedCount)
	s.T().Log("delete count", res.DeletedCount)
}

func TestGuideTestSuite(t *testing.T) {
	suite.Run(t, NewGuideTestSuite("mongodb://127.0.0.1:27017/?compressors=disabled", "test-db"))
}
