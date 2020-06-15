package geo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

// TestMongodbDataSource checks data source by counts
func (s *ResolverTestSuite) TestMongodbDataSource() {
	ctx := context.Background()

	twCounties, err := s.testDatabase.Collection("boundary").Distinct(ctx, "county", bson.M{"country": "Taiwan"})
	s.NoError(err)
	s.Len(twCounties, 22)

	usStates, err := s.testDatabase.Collection("boundary").Distinct(ctx, "state", bson.M{"country": "United States"})
	s.NoError(err)
	s.Len(usStates, 56)

	country, err := s.testDatabase.Collection("boundary").Distinct(ctx, "country", bson.M{})
	s.NoError(err)
	s.Len(country, 198)
}
