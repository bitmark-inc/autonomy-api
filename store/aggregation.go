package store

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/bitmark-inc/autonomy-api/schema"
)

// AggregationMatch helps generate aggregation object for $match
func AggregationMatch(matchCondition bson.M) bson.D {
	match := bson.D{}
	for k, v := range matchCondition {
		match = append(match, bson.E{k, v})
	}

	return bson.D{
		bson.E{"$match", match},
	}
}

// AggregationProject helps generate aggregation object for $project
func AggregationProject(projectCondition bson.M) bson.D {
	project := bson.D{}
	for k, v := range projectCondition {
		project = append(project, bson.E{k, v})
	}

	return bson.D{
		bson.E{"$project", project},
	}
}

// AggregationUnwind helps generate aggregation object for $unwind
func AggregationUnwind(key string) bson.D {
	return bson.D{
		bson.E{"$unwind", key},
	}
}

// AggregationAddFields helps generate aggregation object for $addFields
func AggregationAddFields(fields bson.M) bson.D {
	return bson.D{
		bson.E{
			"$addFields", fields,
		},
	}
}

// AggregationSort helps generate aggregation object for $sort
func AggregationSort(fields bson.M) bson.D {
	return bson.D{
		bson.E{
			"$sort", fields,
		},
	}
}

// AggregationSkip helps generate aggregation object for $skip
func AggregationSkip(number int64) bson.D {
	return bson.D{
		bson.E{
			"$skip", number,
		},
	}
}

// AggregationLimit helps generate aggregation object for $limit
func AggregationLimit(number int64) bson.D {
	return bson.D{
		bson.E{
			"$limit", number,
		},
	}
}

// AggregationGroup helps generate aggregation object for $group
func AggregationGroup(id string, groupConditions bson.D) bson.D {
	group := bson.D{bson.E{"_id", id}}
	group = append(group, groupConditions...)

	return bson.D{
		bson.E{
			"$group", group,
		},
	}
}

// GeoNearOption is an option for AggregationGeoNear help function
type GeoNearOption struct {
	GeoKey             string
	DistanceKey        string
	DistanceMultiplier float64
}

// AggregationGroup helps generate aggregation object for $geoNear
func AggregationGeoNear(coordinates schema.Location, distance int, options ...GeoNearOption) bson.D {
	geoNear := bson.M{
		"near": bson.M{
			"type":        "Point",
			"coordinates": []float64{coordinates.Longitude, coordinates.Latitude},
		},
		"maxDistance": distance,
		"spherical":   true,
	}

	if len(options) > 0 {
		option := options[0]
		if option.GeoKey != "" {
			geoNear["key"] = option.GeoKey
		}

		if option.DistanceKey != "" {
			geoNear["distanceField"] = option.DistanceKey
			if option.DistanceMultiplier != 0 {
				geoNear["distanceMultiplier"] = option.DistanceMultiplier
			}
		}
	}

	return bson.D{
		bson.E{"$geoNear", geoNear},
	}
}
