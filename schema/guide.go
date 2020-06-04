package schema

const TestCenterCollection = "TestCenter"

type TestCenter struct {
	Country         CDSCountryType `json:"country" bson:"country"`
	State           string         `json:"state" bson:"state"`
	County          string         `json:"county" bson:"county"`
	Island          string         `json:"island" bson:"island"`
	InstitutionCode string         `json:"institution_code" bson:"institution_code"`
	Location        GeoJSON        `json:"location" bson:"location"`
	Name            string         `json:"name"  bson:"bson"`
	Address         string         `json:"address" bson:"address"`
	Phone           string         `json:"phone" bson:"phone"`
}
