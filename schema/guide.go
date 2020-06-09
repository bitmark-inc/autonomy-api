package schema

const TestCenterCollection = "TestCenter"

type NearbyTestCenter struct {
	Distance  float64        `json:"distance" bson:"distance"`
	Country   CDSCountryType `json:"country" bson:"country"`
	State     string         `json:"state" bson:"state"`
	County    string         `json:"county" bson:"county"`
	Location  GeoJSON        `json:"-" bson:"location"`
	Latitude  float64        `json:"latitude" bson:"latitude"`
	Longitude float64        `json:"longitude" bson:"longitude"`
	Name      string         `json:"name"  bson:"name"`
	Address   string         `json:"address" bson:"address"`
	Phone     string         `json:"phone" bson:"phone"`
}

type TestCenter struct {
	Country         CDSCountryType `json:"country" bson:"country"`
	State           string         `json:"state" bson:"state"`
	County          string         `json:"county" bson:"county"`
	InstitutionCode string         `json:"institution_code" bson:"institution_code"`
	Location        GeoJSON        `json:"location" bson:"location"`
	Name            string         `json:"name"  bson:"name"`
	Address         string         `json:"address" bson:"address"`
	Phone           string         `json:"phone" bson:"phone"`
}
