package geojson

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type GeoFeature struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   schema.Geometry        `json:"geometry"`
}

type GeoJSON struct {
	Name     string       `json:"name"`
	Features []GeoFeature `json:"features"`
}

func ImportTaiwanBoundary(client *mongo.Client, dbName, geoJSONFile string) error {
	var result GeoJSON

	file, err := os.Open(geoJSONFile)
	if err != nil {
		return err
	}

	if err := json.NewDecoder(file).Decode(&result); err != nil {
		return err
	}

	var boundaries []interface{}
	for _, b := range result.Features {
		county, ok := b.Properties["COUNTYENG"].(string)
		if !ok {
			return fmt.Errorf("invalid county value, %+v", b.Properties["COUNTYENG"])
		}
		boundaries = append(boundaries, schema.Boundary{
			Country:  "Taiwan",
			Island:   "Taiwan",
			State:    "",
			County:   county,
			Geometry: b.Geometry,
		})
	}

	if _, err := client.Database(dbName).Collection(schema.BoundaryCollection).InsertMany(context.Background(), boundaries); err != nil {
		return err
	}

	return nil
}

func ImportUSBoundary(client *mongo.Client, dbName, geoJSONFile string) error {
	var result GeoJSON

	stateAbbrToName := map[string]string{
		"AL": "Alabama",

		"AK": "Alaska",

		"AZ": "Arizona",

		"AR": "Arkansas",

		"CA": "California",

		"CO": "Colorado",

		"CT": "Connecticut",

		"DE": "Delaware",

		"FL": "Florida",

		"GA": "Georgia",

		"HI": "Hawaii",

		"ID": "Idaho",

		"IL": "Illinois",

		"IN": "Indiana",

		"IA": "Iowa",

		"KS": "Kansas",

		"KY": "Kentucky",

		"LA": "Louisiana",

		"ME": "Maine",

		"MD": "Maryland",

		"MA": "Massachusetts",

		"MI": "Michigan",

		"MN": "Minnesota",

		"MS": "Mississippi",

		"MO": "Missouri",

		"MT": "Montana",

		"NE": "Nebraska",

		"NV": "Nevada",

		"NH": "New Hampshire",

		"NJ": "New Jersey",

		"NM": "New Mexico",

		"NY": "New York",

		"NC": "North Carolina",

		"ND": "North Dakota",

		"OH": "Ohio",

		"OK": "Oklahoma",

		"OR": "Oregon",

		"PA": "Pennsylvania",

		"RI": "Rhode Island",

		"SC": "South Carolina",

		"SD": "South Dakota",

		"TN": "Tennessee",

		"TX": "Texas",

		"UT": "Utah",

		"VT": "Vermont",

		"VA": "Virginia",

		"WA": "Washington",

		"WV": "West Virginia",

		"WI": "Wisconsin",

		"WY": "Wyoming",

		"PR": "Puerto Rico",

		"GU": "Guam",

		"VI": "Virgin Islands",

		"MP": "Northern Marianas",

		"DC": "District of Columbia",

		"AS": "American Samoa"}

	file, err := os.Open(geoJSONFile)
	if err != nil {
		return err
	}

	if err := json.NewDecoder(file).Decode(&result); err != nil {
		return err
	}

	var boundaries []interface{}
	for _, b := range result.Features {
		county, ok := b.Properties["namelsad"].(string)
		if !ok {
			return fmt.Errorf("invalid county value, %+v", b.Properties["namelsad"])
		}

		state, ok := b.Properties["stusab"].(string)
		if !ok {
			return fmt.Errorf("invalid state value, %+v", b.Properties["stusab"])
		}

		statename, ok := stateAbbrToName[state]
		if !ok {
			return fmt.Errorf("missing state abbreviation, %+v", state)
		}

		boundaries = append(boundaries, schema.Boundary{
			Country:  "United States",
			Island:   "",
			State:    statename,
			County:   county,
			Geometry: b.Geometry,
		})
	}

	if _, err := client.Database(dbName).Collection(schema.BoundaryCollection).InsertMany(context.Background(), boundaries); err != nil {
		return err
	}

	return nil

}

func ImportWorldCountryBoundary(client *mongo.Client, dbName, geoJSONFile string) error {
	var result GeoJSON

	file, err := os.Open(geoJSONFile)
	if err != nil {
		return err
	}

	if err := json.NewDecoder(file).Decode(&result); err != nil {
		return err
	}

	var boundaries []interface{}
	for _, b := range result.Features {
		country, ok := b.Properties["COUNTRYAFF"].(string)
		if !ok {
			return fmt.Errorf("invalid country value, %+v", b.Properties["COUNTRYAFF"])
		}

		island, ok := b.Properties["COUNTRY"].(string)
		if !ok {
			return fmt.Errorf("invalid island value, %+v", b.Properties["COUNTRY"])
		}

		boundaries = append(boundaries, schema.Boundary{
			Country:  country,
			Island:   island,
			State:    "",
			County:   "",
			Geometry: b.Geometry,
		})
	}

	if _, err := client.Database(dbName).Collection(schema.BoundaryCollection).InsertMany(context.Background(), boundaries); err != nil {
		return err
	}

	return nil
}
