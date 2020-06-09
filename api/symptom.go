package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"

	"github.com/bitmark-inc/autonomy-api/consts"
	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/utils"
	workflow "go.uber.org/cadence/.gen/go/shared"
)

func (s *Server) createSymptom(c *gin.Context) {
	var params schema.Symptom

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	if params.Name == "" {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters)
		return
	}

	id, err := s.mongoStore.CreateSymptom(params)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": id,
	})
	return
}

func (s *Server) getSymptoms(c *gin.Context) {
	a := c.MustGet("account")

	var params struct {
		Language string `form:"lang"`
	}

	if err := c.Bind(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	lang := "en"
	if params.Language != "" {
		lang = params.Language
	}

	account, ok := a.(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}
	var loc *schema.Location
	loc = account.Profile.State.LastLocation
	if nil == loc {
		abortWithEncoding(c, http.StatusBadRequest, errorUnknownAccountLocation)
		return
	}
	symptoms, err := s.mongoStore.ListOfficialSymptoms(lang)
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorUnknownAccountLocation)
	}

	customized, err := s.mongoStore.FindNearbyNonOfficialSymptoms(consts.NEARBY_DISTANCE_RANGE, *loc)
	if err != nil {
		c.Error(err)
	}

	if customized != nil {
		symptoms = append(symptoms, customized...)
	}

	c.JSON(http.StatusOK, gin.H{"symptoms": symptoms})
}

func (s *Server) getSymptomsV2(c *gin.Context) {
	a := c.MustGet("account")

	var params struct {
		Language string `form:"lang"`
		All      bool   `form:"all"`
	}

	if err := c.Bind(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	lang := "en"
	if params.Language != "" {
		lang = params.Language
	}

	account, ok := a.(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}
	var loc *schema.Location
	loc = account.Profile.State.LastLocation
	if nil == loc {
		abortWithEncoding(c, http.StatusBadRequest, errorUnknownAccountLocation)
		return
	}

	official, err := s.mongoStore.ListOfficialSymptoms(lang)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	if params.All {
		suggested, err := s.mongoStore.ListSuggestedSymptoms(lang)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
			return
		}

		customized, err := s.mongoStore.ListCustomizedSymptoms()
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"official_symptoms":   official,
			"suggested_symptoms":  suggested,
			"customized_symptoms": customized,
		})
		return
	}

	customized, err := s.mongoStore.FindNearbyNonOfficialSymptoms(consts.NEARBY_DISTANCE_RANGE, *loc)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"official_symptoms":     official,
		"neighborhood_symptoms": customized,
	})
}

func (s *Server) reportSymptoms(c *gin.Context) {
	a := c.MustGet("account")
	account, ok := a.(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	loc := account.Profile.State.LastLocation
	if nil == loc {
		abortWithEncoding(c, http.StatusBadRequest, errorUnknownAccountLocation)
		return
	}

	var params struct {
		Symptoms []string `json:"symptoms"`
	}

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	if 0 == len(params.Symptoms) {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, errors.New(errorMessageMap[1010]))
		return
	}
	symptoms, err := s.mongoStore.FindSymptomsByIDs(params.Symptoms)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	data := schema.SymptomReportData{
		ProfileID:     account.Profile.ID.String(),
		AccountNumber: account.Profile.AccountNumber,
		Symptoms:      symptoms,
		Location:      schema.GeoJSON{Type: "Point", Coordinates: []float64{loc.Longitude, loc.Latitude}},
		Timestamp:     time.Now().UTC().Unix(),
	}
	if err := s.mongoStore.SymptomReportSave(&data); err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	if _, err := s.mongoStore.SyncProfileIndividualMetrics(account.Profile.ID.String()); err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	official, customied := schema.SplitSymptoms(symptoms)
	if len(customied) > 0 {
		err = s.mongoStore.UpdateAreaProfileSymptom(customied, *loc)
		if err != nil { // do nothing
			c.Error(err)
		}
	}

	accts, err := s.mongoStore.NearestDistance(consts.NEARBY_DISTANCE_RANGE, *loc)
	if nil == err {
		go func() {
			if err := utils.TriggerAccountUpdate(*s.cadenceClient, c, accts); err != nil {
				sentry.CaptureException(err)
			}
		}()
	} else {
		c.Error(err)
	}
	pois, err := s.mongoStore.NearestPOI(consts.NEARBY_DISTANCE_RANGE, *loc)
	if nil == err {
		go func() {
			if err := utils.TriggerPOIUpdate(*s.cadenceClient, c, pois); err != nil {
				sentry.CaptureException(err)
			}
		}()

		go func() {
			if err := utils.TriggerAccountSymptomFollowUpNudge(*s.cadenceClient, c, account.AccountNumber); err != nil {
				if _, ok := err.(*workflow.WorkflowExecutionAlreadyStartedError); !ok {
					sentry.CaptureException(err)
				}
			}

			if err := utils.TriggerAccountHighRiskFollowUpNudge(*s.cadenceClient, c, account.AccountNumber); err != nil {
				if _, ok := err.(*workflow.WorkflowExecutionAlreadyStartedError); !ok {
					sentry.CaptureException(err)
				}
			}
		}()
	} else {
		c.Error(err)
	}
	if len(official) <= 0 {
		c.JSON(http.StatusOK, gin.H{"result": "OK", "official": 0, "guide": []schema.NearbyTestCenter{}})
		return
	}
	centers, err := s.mongoStore.NearbyTestCenter(*loc, 10)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusOK, gin.H{"result": "OK", "official": len(official), "guide": []schema.NearbyTestCenter{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": "OK", "official": len(official), "guide": centers})
	return

}
