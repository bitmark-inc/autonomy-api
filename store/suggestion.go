package store

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type Suggestion interface {
	ListSuggestedResources(profileID, lang string) ([]schema.Resource, error)
}

// ListSuggestedResources returns resources suggestion for searching places. It first collect all
// resources by:
// 1. resources a user has rated
// 2. pre-defined resources that are marked important
func (m *mongoDB) ListSuggestedResources(profileID, lang string) ([]schema.Resource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	cursor, err := c.Aggregate(ctx, mongo.Pipeline{
		AggregationMatch(bson.M{"id": profileID}),
		AggregationProject(bson.M{
			"id": 1,
			"points_of_interest.resource_ratings.resources": 1,
		}),
		AggregationUnwind("$points_of_interest"),
		AggregationUnwind("$points_of_interest.resource_ratings.resources"),
		AggregationAddFields(bson.M{
			"resource": bson.A{
				bson.A{
					"$points_of_interest.resource_ratings.resources.resource.id",
					"$points_of_interest.resource_ratings.resources.resource",
				},
			},
		}),
		AggregationProject(bson.M{
			"id": 1,
			"resource": bson.M{
				"$arrayToObject": "$resource",
			},
		}),
		AggregationGroup("$id", bson.D{
			bson.E{
				Key: "resources",
				Value: bson.M{
					"$mergeObjects": "$resource",
				},
			},
		}),
	})

	if err != nil {
		return nil, err
	}

	var result []struct {
		Resources map[string]schema.Resource `json:"resources" bson:"resources"`
	}

	if err := cursor.All(ctx, &result); nil != err {
		return nil, err
	}

	importantResources, err := getResourceList(lang, true)
	if err != nil {
		return nil, err
	}

	var resources []schema.Resource
	if len(result) == 1 {
		selfOwnedResources := result[0].Resources
		resources = make([]schema.Resource, 0, len(result[0].Resources))
		for id, r := range selfOwnedResources {
			if name, _ := ResolveResourceNameByID(id, lang); name != "" {
				r.Name = name
			}
			resources = append(resources, r)
		}

		for _, r := range importantResources {
			if _, ok := selfOwnedResources[r.ID]; !ok {
				resources = append(resources, r)
			}
		}
	} else {
		resources = importantResources
	}

	return resources, nil
}
