package store

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/bitmark-inc/autonomy-api/schema"
)

var (
	ErrPOINotFound = fmt.Errorf("poi not found")
)

type POI interface {
	AddPOI(accountNumber string, alias, address string, lon, lat float64) (*schema.POI, error)
	GetPOI(accountNumber string) ([]*schema.POIDetail, error)
	UpdatePOIAlias(accountNumber, alias string, poiID primitive.ObjectID) error
	DeletePOI(accountNumber string, poiID primitive.ObjectID) error
}

// AddPOI inserts a new POI record if it doesn't exist and append it to user's profile
func (m *mongoDB) AddPOI(accountNumber string, alias, address string, lon, lat float64) (*schema.POI, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.POICollection)

	var poi schema.POI
	query := bson.M{
		"location.coordinates.0": lon,
		"location.coordinates.1": lat,
	}
	if err := c.FindOne(ctx, query).Decode(&poi); err != nil {
		if err == mongo.ErrNoDocuments {
			poi = schema.POI{
				Location: &schema.GeoJSON{
					Type:        "Point",
					Coordinates: []float64{lon, lat},
				},
			}
			result, err := c.InsertOne(ctx, bson.M{"location": poi.Location})
			if err != nil {
				return nil, err
			}
			poi.ID = result.InsertedID.(primitive.ObjectID)
		} else {
			return nil, err
		}
	}

	poiDesc := &schema.POIDesc{
		ID:      poi.ID,
		Alias:   alias,
		Address: address,
	}
	if err := m.AppendPOIForAccount(accountNumber, poiDesc); err != nil {
		return nil, err
	}

	return &poi, nil
}

// GetPOI finds the POI list of an account along with customied alias and address
func (m *mongoDB) GetPOI(accountNumber string) ([]*schema.POIDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollectionName)

	// find user's POI list
	var result struct {
		Points []*schema.POIDetail `bson:"points_of_interest"`
	}
	query := bson.M{"account_number": accountNumber}
	if err := c.FindOne(ctx, query).Decode(&result); err != nil {
		return nil, err
	}
	if result.Points == nil { // user hasn't tracked any location yet
		return []*schema.POIDetail{}, nil
	}

	// find scores
	poiIDs := make([]primitive.ObjectID, 0)
	for _, p := range result.Points {
		poiIDs = append(poiIDs, p.ID)
	}

	// $in query doesn't guarantee order
	// use aggregation to sort the nested docs according to the query order
	pipeline := []bson.M{
		{"$match": bson.M{"_id": bson.M{"$in": poiIDs}}},
		{"$addFields": bson.M{"__order": bson.M{"$indexOfArray": bson.A{poiIDs, "$_id"}}}},
		{"$sort": bson.M{"__order": 1}},
	}
	c = m.client.Database(m.database).Collection(schema.POICollection)
	cursor, err := c.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var pois []*schema.POI
	if err = cursor.All(ctx, &pois); err != nil {
		return nil, err
	}

	if len(pois) != len(result.Points) {
		return nil, fmt.Errorf("poi data wrongly retrieved or removed")
	}

	for i, p := range result.Points {
		p.Score = pois[i].Score
		p.Location.Longitude = pois[i].Location.Coordinates[0]
		p.Location.Latitude = pois[i].Location.Coordinates[1]
	}

	return result.Points, nil
}

// UpdatePOIAlias updates the alias of a POI for the specified account
func (m *mongoDB) UpdatePOIAlias(accountNumber, alias string, poiID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollectionName)
	query := bson.M{
		"account_number":        accountNumber,
		"points_of_interest.id": poiID,
	}
	update := bson.M{"$set": bson.M{"points_of_interest.$.alias": alias}}
	result, err := c.UpdateOne(ctx, query, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrPOINotFound
	}

	return nil
}

func (m *mongoDB) DeletePOI(accountNumber string, poiID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollectionName)
	query := bson.M{
		"account_number":        accountNumber,
		"points_of_interest.id": poiID,
	}
	update := bson.M{"$pull": bson.M{"points_of_interest": bson.M{"id": poiID}}}
	if _, err := c.UpdateOne(ctx, query, update); err != nil {
		return err
	}

	return nil
}
