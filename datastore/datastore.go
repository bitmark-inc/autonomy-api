package datastore

import (
	"encoding/json"
	"fmt"
	"net/http/httputil"

	"github.com/bitmark-inc/autonomy-api/external/datastore"
	"github.com/bitmark-inc/autonomy-api/schema"
	log "github.com/sirupsen/logrus"
)

type DefaultPOIResource string

const (
	FACE_COVERINGS_REQUIRED               DefaultPOIResource = "face_coverings_required"
	SOCIAL_DISTANCING                     DefaultPOIResource = "social_distancing"
	TEMPERATURE_CHECKS                    DefaultPOIResource = "temperature_checks"
	HAND_SANITIZER                        DefaultPOIResource = "hand_sanitizer"
	EQUIPMENT_DISINFECTED                 DefaultPOIResource = "equipment_disinfected"
	SURFACES_DISINFECTED                  DefaultPOIResource = "surfaces_disinfected"
	HAND_WASHING_FACILITIES               DefaultPOIResource = "hand_washing_facilities"
	GOOD_AIR_CIRCULATION                  DefaultPOIResource = "good_air_circulation"
	OUTDOOR_OPTIONS                       DefaultPOIResource = "outdoor_options"
	SPECIAL_HOURS_FOR_AT_RISK_POPULATIONS DefaultPOIResource = "special_hours_for_at_risk_populations"
)

type DataStore struct {
	pds *datastore.PersonalDataStore
	cds *datastore.CommunityDataStore
}

func NewDataStore(pdsEndpoint, cdsEndpoint string) *DataStore {
	return &DataStore{
		pds: datastore.NewPersonalDataStore(pdsEndpoint),
		cds: datastore.NewCommunityDataStore(cdsEndpoint),
	}
}

func (d *DataStore) GetPOIRating(token, id string) (*schema.POIRating, error) {
	r, err := d.pds.GetRating(token, id)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		b, _ := httputil.DumpResponse(r, true)
		log.WithField("resp", string(b)).Debug("error from ratings api")
		return nil, fmt.Errorf("fail to get ratings")
	}

	var rating schema.POIRating

	if err := json.NewDecoder(r.Body).Decode(&rating); err != nil {
		return nil, err
	}

	rating.ID = id

	return &rating, nil
}

func (d *DataStore) SetPOIRating(token, id string, ratings map[string]float64) error {
	r, err := d.pds.SetRating(token, id, ratings)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		b, _ := httputil.DumpResponse(r, true)
		log.WithField("resp", string(b)).Debug("error from ratings api")
		return fmt.Errorf("fail to set ratings")
	}

	return nil
}

func (d *DataStore) GetPOICommunityRatings(token string, ids []string) (map[string]schema.POISummarizedRating, error) {
	r, err := d.cds.GetPOIRatings(token, ids)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		b, _ := httputil.DumpResponse(r, true)
		log.WithField("resp", string(b)).Debug("error from ratings api")
		return nil, fmt.Errorf("fail to set ratings")
	}

	var ratings map[string]schema.POISummarizedRating

	if err := json.NewDecoder(r.Body).Decode(&ratings); err != nil {
		return nil, err
	}

	return ratings, nil
}

func (d *DataStore) SetPOICommunityRating(token string, id string, ratings map[string]float64) error {
	r, err := d.cds.SetPOIRating(token, id, ratings)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		b, _ := httputil.DumpResponse(r, true)
		log.WithField("resp", string(b)).Debug("error from ratings api")
		return fmt.Errorf("fail to set ratings")
	}

	return nil
}

func (d *DataStore) GetCommunitySymptomReportItems(token string, days int) (*schema.ReportItems, error) {
	r, err := d.cds.GetSymptomReportItems(token, days)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		b, _ := httputil.DumpResponse(r, true)
		log.WithField("resp", string(b)).Debug("error from report-items api")
		return nil, fmt.Errorf("fail to get symptom report items")
	}

	var items schema.ReportItems

	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		return nil, err
	}

	return &items, nil
}
