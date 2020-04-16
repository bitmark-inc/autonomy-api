package geoinfo

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"googlemaps.github.io/maps"
)

const (
	logPrefix      = "geoinfo"
	defaultTimeout = 5 * time.Second
)

// GeoInfo - interface to operate google maps
type GeoInfo interface {
	Get(string) ([]maps.GeocodingResult, error)
}

type geoInfo struct {
	client *maps.Client
}

// latLng - a string representation of a Lat,Lng pair, e.g. 1.23,4.56
func (g geoInfo) Get(latLngStr string) ([]maps.GeocodingResult, error) {
	log.WithFields(log.Fields{
		"prefix":         logPrefix,
		"lat lng string": latLngStr,
	}).Debug("query geo information")

	latlng, err := maps.ParseLatLng(latLngStr)
	if err != nil {
		log.WithFields(log.Fields{
			"prefix":         logPrefix,
			"lat lng string": latLngStr,
			"error":          err,
		}).Error("parse lat lng string")
		return []maps.GeocodingResult{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	return g.client.Geocode(ctx, &maps.GeocodingRequest{LatLng: &latlng})
}

// New - new GeoInfo interface
func New(apiKey string) (GeoInfo, error) {
	client, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		log.WithFields(log.Fields{
			"prefix": logPrefix,
			"error":  err,
		}).Error("new map client")

		return nil, err
	}

	return &geoInfo{
		client: client,
	}, nil
}
