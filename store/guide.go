package store

import (
	"context"
	"fmt"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bitmark-inc/autonomy-api/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Guide interface {
	NearbyTestCenter(loc schema.Location, limit int64) ([]schema.NearbyTestCenter, error)
}

func (m mongoDB) NearbyTestCenter(loc schema.Location, limit int64) ([]schema.NearbyTestCenter, error) {
	c := m.client.Database(m.database).Collection(schema.TestCenterCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	pipeline := mongo.Pipeline{
		geoWithDistanceAggregate(loc),
		//matchAggregate(loc.Country),
		limitAggregate(limit),
		bson.D{{"$project", bson.M{
			"_id":      -1,
			"distance": 1,
			"center": bson.M{
				"country":          "$country",
				"institution_code": "$institution_code",
				"location":         "$location",
				"name":             "$name",
				"address":          "$address",
				"phone":            "$phone",
			},
		}}},
	}

	opts := options.Aggregate().SetMaxTime(5 * time.Second)
	cursor, err := c.Aggregate(ctx, pipeline, opts)
	if err != nil {
		log.WithError(err).Warnf("can not aggregate nearbyTest center")
		return []schema.NearbyTestCenter{}, err
	}
	results := []schema.NearbyTestCenter{}
	for cursor.Next(ctx) {
		var center schema.NearbyTestCenter
		if err := cursor.Decode(&center); err != nil {
			log.WithError(err).Warnf("nearbyTest center decode fail")
			continue
		}
		km, err := strconv.ParseFloat(fmt.Sprintf("%0.2f", center.Distance/1000), 64)
		if err != nil {
			log.WithError(err).Warnf("TestCenter Parse float error")
			continue
		}
		center.Distance = km
		results = append(results, center)
		log.Info(center)
	}
	return results, nil
}

func matchAggregate(locCountry string) bson.D {
	return bson.D{{"$match", bson.M{"country": locCountry}}}
}

func geoWithDistanceAggregate(loc schema.Location) bson.D {
	return bson.D{{"$geoNear", bson.M{
		"near":          bson.M{"type": "Point", "coordinates": bson.A{loc.Longitude, loc.Latitude}},
		"distanceField": "distance",
		"spherical":     true,
	}}}
}

func limitAggregate(number int64) bson.D {
	return bson.D{{"$limit", number}}
}
