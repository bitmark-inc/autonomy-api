package nominatim

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

type QueryResult struct {
	PlaceID     int     `json:"place_id"`
	OSMType     string  `json:"osm_type"`
	OSMID       int     `json:"osm_id"`
	Latitude    float64 `json:"lat,string"`
	Longitude   float64 `json:"lon,string"`
	DisplayName string  `json:"display_name"`
	Class       string  `json:"class"`
	Type        string  `json:"type"`
}

type NominatimClient struct {
	endpoint string
	client   *http.Client
}

func New(endpoint string) *NominatimClient {
	return &NominatimClient{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (n *NominatimClient) Query(query string) ([]QueryResult, error) {
	q := url.URL{
		Path: "search.php",
		RawQuery: url.Values{
			"q":      []string{query},
			"format": []string{"json"},
		}.Encode(),
	}

	reqString := fmt.Sprintf("%s/%s", n.endpoint, q.String())
	log.WithField("prefix", "nominatim").WithField("req", reqString).Debug("request from nominatim")

	resp, err := n.client.Get(reqString)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Print out the response in console log
	dumpBytes, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.WithField("prefix", "nominatim").WithError(err).Error("fail to dump response")
	}

	if resp.StatusCode != http.StatusOK {
		log.WithField("prefix", "nominatim").WithField("resp", string(dumpBytes)).Error("error response from nominatim")
		return nil, fmt.Errorf("fail to query address")
	}

	log.WithField("prefix", "nominatim").WithField("resp", string(dumpBytes)).Debug("response from nominatim")

	var result []QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
