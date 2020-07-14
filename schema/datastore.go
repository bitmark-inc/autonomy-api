package schema

type POIRating struct {
	ID            string             `json:"id"`
	RatingAverage *float64           `json:"rating_avg,omitempty"`
	Ratings       map[string]float64 `json:"ratings"`
}

type ReportItems struct {
	ReportItems []struct {
		ID           string         `json:"id"`
		Name         string         `json:"name"`
		Value        *int           `json:"value"`
		ChangeRate   *float64       `json:"change_rate"`
		Distribution map[string]int `json:"distribution"`
	} `json:"report_items"`
}
