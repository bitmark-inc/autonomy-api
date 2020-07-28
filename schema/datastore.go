package schema

type POIRating struct {
	ID            string             `json:"id"`
	RatingAverage *float64           `json:"rating_avg,omitempty"`
	Ratings       map[string]float64 `json:"ratings"`
}

type RatingInfo struct {
	Score  float64 `bson:"score" json:"score"`
	Counts int     `bson:"counts" json:"counts"`
}

type POISummarizedRating struct {
	ID            string                `json:"id"`
	LastUpdated   int64                 `json:"last_updated"`
	RatingAverage float64               `json:"rating_avg"`
	RatingCounts  int64                 `json:"rating_counts"`
	Ratings       map[string]RatingInfo `json:"ratings"`
}

type ReportItems struct {
	CheckinsNumPastThreeDays int `json:"checkins_num_past_three_days"`
	ReportItems              []struct {
		ID           string         `json:"id"`
		Name         string         `json:"name"`
		Value        *int           `json:"value"`
		ChangeRate   *float64       `json:"change_rate"`
		Distribution map[string]int `json:"distribution"`
	} `json:"report_items"`
}
