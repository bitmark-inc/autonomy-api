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
	return result.ResourceRatings, nil
}

func (m *mongoDB) UpdatePOIRatingMetric(accountNumber string, poiID primitive.ObjectID, ratings []schema.RatingResource) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	c := m.client.Database(m.database).Collection(schema.POICollection)
	poiMetric, err := m.GetPOIResourceMetric(poiID)
	if err != nil {
		log.WithFields(log.Fields{
			"prefix": mongoLogPrefix,
			"poi ID": poiID.String(),
			"error":  err,
		}).Error("get poi fail")
		return err
	}

	profileMetric, err := m.GetProfilesRatingMetricByPOI(accountNumber, poiID)

	log.Info("profileMetric:", profileMetric)
	if err != nil && err != ErrPOINotFound {
		log.WithFields(log.Fields{
			"prefix":        mongoLogPrefix,
			"poi ID":        poiID.String(),
			"acount number": accountNumber,
			"error":         err,
		}).Error("get poi fail")
		return err
	}

	now := time.Now().UTC()
	todayStartAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	poiResourceMap := make(map[string]schema.POIResourceRating)
	for _, r := range poiMetric.Resources { // make a current poi resources map
		old, ok := poiResourceMap[r.Resource.ID]
		if ok {
			if old.LastUpdate < todayStartAt.Unix() {
				r.LastDayScore = old.Score
				r.LastDayRatings = old.Ratings
			}
		}
		r.LastUpdate = now.Unix()
		poiResourceMap[r.Resource.ID] = r
	}
	profileResourceMap := make(map[string]schema.RatingResource)

	for _, r := range profileMetric.Resources { // make a current profile resource map
		profileResourceMap[r.ID] = r
	}
	for _, r := range ratings { //add user rating (could be an empty one) to score
		existPOIRating, ok := poiResourceMap[r.ID] // check on poi metric,
		if !ok {                                   // all resources in profile should be in the poi
			return ErrPOINotFound
		}
		existProfileRating, ok := profileResourceMap[r.ID]
		update := false
		if ok { // update the score
			update = true
		} else {
			existProfileRating = schema.RatingResource{}
		}
		count, sum, average := score.ResourceScore(existPOIRating, r, existProfileRating, update)

		//	count, sum, average := score.ResourceScore(val.Ratings, val.SumOfScore, r,)

		poiResourceMap[r.Resource.ID] = schema.POIResourceRating{
			Resource:       r.Resource,
			SumOfScore:     sum,
			Score:          average,
			Ratings:        count,
			LastUpdate:     existPOIRating.LastUpdate,
			LastDayScore:   existPOIRating.LastDayScore,
			LastDayRatings: existPOIRating.LastDayRatings,
		}
	}

	poiRatings := []schema.POIResourceRating{}

	for _, r := range poiResourceMap {
		poiRatings = append(poiRatings, r)
	}

	query := bson.M{
		"_id": poiID,
	}
	update := bson.M{
		"$set": bson.M{
			"resource_ratings": schema.POIRatingsMetric{
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
		}).Error("update poi resource_ratings")
		return err
	}

	if result.MatchedCount == 0 {
		log.WithFields(log.Fields{
			"prefix": mongoLogPrefix,
			"poi ID": pid,
			"error":  ErrPOINotFound.Error(),
		}).Error("update poi resource_ratings")
		return ErrPOINotFound
	}
	profileMetric.LastUpdate = time.Now().Unix()
	profileMetric.Resources = ratings
	err = m.UpdateProfilePOIRatingMetric(accountNumber, poiID, profileMetric)
	if err != nil {
		return err
	}
	return nil
}
