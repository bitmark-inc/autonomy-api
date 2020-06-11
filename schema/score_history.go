package schema

const (
	ScoreHistoryCollection = "scoreHistory"
)

type ScoreRecordType string

const (
	ScoreRecordTypeIndividual = "individual"
	ScoreRecordTypePOI        = "poi"
)

type ScoreRecord struct {
	Owner string          `bson:"owner"`
	Type  ScoreRecordType `bson:"type"`
	Score float64         `bson:"score"`
	Date  string          `bson:"date"`
}
