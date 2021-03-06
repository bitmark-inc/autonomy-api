package schema

type CDSCountryType string

const (
	CdsUSA     = "United States"
	CdsTaiwan  = "Taiwan"
	CdsIceland = "Iceland"
)

var CDSCountyCollectionMatrix = map[CDSCountryType]string{
	CDSCountryType(CdsUSA):     "ConfirmUS",
	CDSCountryType(CdsTaiwan):  "ConfirmTaiwan",
	CDSCountryType(CdsIceland): "ConfirmIceland",
}

type CDSData struct {
	Name           string   `json:"name" bson:"name"`
	City           string   `json:"city" bson:"city"`
	County         string   `json:"county" bson:"county"`
	State          string   `json:"state" bson:"state"`
	Country        string   `json:"country" bson:"country"`
	Level          string   `json:"level" bson:"level"`
	Cases          float64  `json:"cases" bson:"cases"`
	Deaths         float64  `json:"deaths" bson:"deaths"`
	Recovered      float64  `json:"recovered" bson:"recovered"`
	Active         float64  `json:"active" bson:"active"`
	ReportTime     int64    `json:"report_ts" bson:"report_ts"`
	UpdateTime     int64    `json:"update_ts"  bson:"update_ts"`
	ReportTimeDate string   `json:"report_date" bson:"report_date"`
	CountryID      string   `json:"countryId" bson:"countryId"`
	StateID        string   `json:"stateId" bson:"stateId"`
	CountyID       string   `json:"countyId" bson:"countyId"`
	Location       GeoJSON  `json:"location" bson:"location"`
	Timezone       []string `json:"tz" bson:"tz"`
}

type CDSScoreDataSet struct {
	Name  string  `json:"name" bson:"name"`
	Cases float64 `json:"cases" bson:"cases"`
}
