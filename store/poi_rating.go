package store

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/score"
)

// poi_rating.go is an extension of  poi.go
// methods of interface are defined in poi.go
func (m *mongoDB) GetPOIResourceMetric(poiID primitive.ObjectID) (schema.POIRatingsMetric, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	c := m.client.Database(m.database).Collection(schema.POICollection)
	filter := bson.M{
		"_id": poiID,
	}
	var result schema.POI
	err := c.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		log.WithFields(log.Fields{
			"prefix": mongoLogPrefix,
			"poi ID": poiID.String(),
			"error":  err,
		}).Error("get poi fail")
		return schema.POIRatingsMetric{}, err
	}
	log.WithFields(log.Fields{
		"prefix": mongoLogPrefix,
		"poi ID": poiID.String(),
		"result": result,
	}).Info("get poi")

	return result.ResourceRatings, nil
}

func (m *mongoDB) UpdatePOIRatingMetric(poiID primitive.ObjectID, ratings []schema.RatingResource) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	c := m.client.Database(m.database).Collection(schema.POICollection)

	metric, err := m.GetPOIResourceMetric(poiID)

	if err != nil {
		log.WithFields(log.Fields{
			"prefix": mongoLogPrefix,
			"poi ID": poiID.String(),
			"error":  err,
		}).Error("get poi fail")
		return err
	}
	now := time.Now().UTC()
	todayStartAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	resourceMap := make(map[string]schema.POIResourceRating)
	for _, r := range metric.Resources { // make a current resources map
		old, ok := resourceMap[r.Resource.ID]
		if ok {
			if old.LastUpdate < todayStartAt.Unix() {
				r.LastDayScore = old.Score
				r.LastDayRatings = old.Ratings
			}
		}
		r.LastUpdate = now.Unix()
		resourceMap[r.Resource.ID] = r
	}

	for _, r := range ratings { //add user rating (could be an empty one) to score
		val, ok := resourceMap[r.Resource.ID]

		if !ok {
			val = schema.POIResourceRating{}
		}

		count, sum, average := score.ResourceScore(val.Ratings, val.SumOfScore, r)
		resourceMap[r.Resource.ID] = schema.POIResourceRating{
			Resource:       r.Resource,
			SumOfScore:     sum,
			Score:          average,
			Ratings:        count,
			LastUpdate:     val.LastUpdate,
			LastDayScore:   val.LastDayScore,
			LastDayRatings: val.LastDayRatings,
		}
		log.Info(resourceMap[r.Resource.ID])
	}

	poiRatings := []schema.POIResourceRating{}
	for _, r := range resourceMap {
		poiRatings = append(poiRatings, r)
	}

	query := bson.M{
		"_id": poiID,
	}
	update := bson.M{
		"$set": bson.M{
			"rating_metric": schema.POIRatingsMetric{
				Resources:  poiRatings,
				LastUpdate: now.Unix(),
			},
		},
	}

	result, err := c.UpdateOne(ctx, query, update)
	pid := poiID.String()
	if err != nil {
		log.WithFields(log.Fields{
			"prefix": mongoLogPrefix,
			"poi ID": pid,
			"error":  err,
		}).Error("update poi rating_ metric")
		return err
	}

	if result.MatchedCount == 0 {
		log.WithFields(log.Fields{
			"prefix": mongoLogPrefix,
			"poi ID": pid,
			"error":  ErrPOINotFound.Error(),
		}).Error("update poi rating_metric")
		return ErrPOINotFound
	}

	return nil
}
