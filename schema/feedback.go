package schema

const (
	FeedbackCollection = "feedback"
)

type Feedback struct {
	ID            string `json:"id" bson:"_id,omitempty"`
	AccountNumber string `json:"account_number" bson:"account_number"`
	UserSatisfied bool   `json:"user_satisfied" bson:"user_satisfied"`
	Feedback      string `json:"feedback" bson:"feedback"`
}
