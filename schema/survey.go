package schema

const (
	SurveyCollection = "survey"
)

type Survey struct {
	ID            string      `json:"id" bson:"_id,omitempty"`
	AccountNumber string      `json:"account_number" bson:"account_number"`
	SurveyID      string      `json:"survey_id" bson:"survey_id"`
	Contents      interface{} `json:"contents" bson:"contents"`
}
