package store

import "go.mongodb.org/mongo-driver/bson"

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
