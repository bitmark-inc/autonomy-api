package schema

type AggregationTimeGranularity string

const (
	AggregationByMonth AggregationTimeGranularity = "month"
	AggregationByDay   AggregationTimeGranularity = "day"
)

type BucketAggregation struct {
	ID      string   `bson:"_id"`
	Buckets []Bucket `bson:"buckets"`
}

type Bucket struct {
	Name  string `bson:"name"`
	Value int    `bson:"value"`
}
