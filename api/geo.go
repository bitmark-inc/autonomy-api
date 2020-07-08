package api

import (
	"fmt"
	"strconv"
	"strings"
)

// parseGeoPosition will parse latitude and longitude from the geo-position string
func parseGeoPosition(geoPosition string) (float64, float64, error) {
	positions := strings.Split(geoPosition, ";")

	if len(positions) != 2 {
		return 0, 0, fmt.Errorf("invalid geo-position value")
	}

	lat, err := strconv.ParseFloat(positions[0], 64)
	if err != nil {

		return 0, 0, err
	}

	long, err := strconv.ParseFloat(positions[1], 64)
	if err != nil {

		return 0, 0, err
	}

	return lat, long, nil
}
