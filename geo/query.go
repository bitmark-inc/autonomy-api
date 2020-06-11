package geo

import (
	"fmt"

	"github.com/bitmark-inc/autonomy-api/external/nominatim"
)

var (
	ErrLocationNotFound       = fmt.Errorf("location is not found")
	ErrSearcherNotInitialized = fmt.Errorf("location searcher is not initialized")
)

type LocationSearcher interface {
	LookupCoordinate(string) (float64, float64, error)
}

type NominatimSearcher struct {
	client *nominatim.NominatimClient
}

func NewNominatimSearcher(endpoint string) *NominatimSearcher {
	return &NominatimSearcher{
		client: nominatim.New(endpoint),
	}
}

func (n *NominatimSearcher) LookupCoordinate(query string) (float64, float64, error) {
	results, err := n.client.Query(query)
	if err != nil {
		return 0, 0, err
	}

	if len(results) == 0 {
		return 0, 0, ErrLocationNotFound
	}

	return results[0].Latitude, results[0].Longitude, nil
}

var defaultSearcher LocationSearcher

func SetLocationSearcher(searcher LocationSearcher) {
	defaultSearcher = searcher
}

func LookupCoordinate(query string) (float64, float64, error) {
	if defaultSearcher == nil {
		return 0, 0, ErrSearcherNotInitialized
	}

	return defaultSearcher.LookupCoordinate(query)
}
