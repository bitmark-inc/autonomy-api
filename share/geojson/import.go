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
		"AK": "Alaska", "AL": "Alabama", "AR": "Arkansas", "AS": "American Samoa", "AZ": "Arizona",
		"CA": "California", "CO": "Colorado", "CT": "Connecticut",
		"DC": "District of Columbia", "DE": "Delaware",
		"FL": "Florida", "GA": "Georgia", "GU": "Guam", "HI": "Hawaii",
		"IA": "Iowa", "ID": "Idaho", "IL": "Illinois", "IN": "Indiana",
		"KS": "Kansas", "KY": "Kentucky", "LA": "Louisiana",
		"MA": "Massachusetts", "MD": "Maryland", "ME": "Maine", "MI": "Michigan", "MN": "Minnesota",
		"MO": "Missouri", "MP": "Northern Marianas", "MS": "Mississippi", "MT": "Montana",
		"NC": "North Carolina", "ND": "North Dakota", "NE": "Nebraska", "NH": "New Hampshire",
		"NJ": "New Jersey", "NM": "New Mexico", "NV": "Nevada", "NY": "New York",
		"OH": "Ohio", "OK": "Oklahoma", "OR": "Oregon", "PA": "Pennsylvania", "PR": "Puerto Rico",
		"RI": "Rhode Island", "SC": "South Carolina", "SD": "South Dakota",
		"TN": "Tennessee", "TX": "Texas", "UT": "Utah",
		"VA": "Virginia", "VI": "Virgin Islands", "VT": "Vermont",
		"WA": "Washington", "WI": "Wisconsin", "WV": "West Virginia", "WY": "Wyoming",
	}

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
