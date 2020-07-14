package datastore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DataStoreClient struct {
	client *http.Client
}

func NewDataStoreClient() *DataStoreClient {
	return &DataStoreClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *DataStoreClient) MakeRequest(request *http.Request, token string) (*http.Response, error) {
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	if request.Header.Get("Content-Type") == "" {
		request.Header.Add("Content-Type", "application/json")
	}
	return c.client.Do(request)
}

type PersonalDataStore struct {
	apiEndpoint string
	client      *DataStoreClient
}

func NewPersonalDataStore(endpoint string) *PersonalDataStore {
	u, _ := url.Parse(endpoint)

	apiEndpoint := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
	}

	return &PersonalDataStore{
		apiEndpoint: apiEndpoint.String(),
		client:      NewDataStoreClient(),
	}
}

func (pds *PersonalDataStore) GetRating(token, poiID string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/poi_rating/%s", pds.apiEndpoint, poiID), nil)
	return pds.client.MakeRequest(req, token)
}

func (pds *PersonalDataStore) SetRating(token, poiID string, ratings map[string]float64) (*http.Response, error) {
	// prepare request body
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(map[string]interface{}{
		"ratings": ratings,
	}); err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/poi_rating/%s", pds.apiEndpoint, poiID), &body)
	return pds.client.MakeRequest(req, token)
}

type CommunityDataStore struct {
	apiEndpoint string
	client      *DataStoreClient
}

func NewCommunityDataStore(endpoint string) *CommunityDataStore {
	u, _ := url.Parse(endpoint)

	apiEndpoint := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
	}

	return &CommunityDataStore{
		apiEndpoint: apiEndpoint.String(),
		client:      NewDataStoreClient(),
	}
}

func (cds *CommunityDataStore) SetPOIRating(token, poiID string, ratings map[string]float64) (*http.Response, error) {
	// prepare request body
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(map[string]interface{}{
		"ratings": ratings,
	}); err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/poi_rating/%s", cds.apiEndpoint, poiID), &body)
	return cds.client.MakeRequest(req, token)
}

func (cds *CommunityDataStore) GetPOIRating(token, poiID string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/poi_rating/%s", cds.apiEndpoint, poiID), nil)
	return cds.client.MakeRequest(req, token)
}

func (cds *CommunityDataStore) GetPOIRatings(token string, poiIDs []string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/poi_rating?poi_ids=%s", cds.apiEndpoint, strings.Join(poiIDs, ",")), nil)
	return cds.client.MakeRequest(req, token)
}
