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

	customized, err := s.mongoStore.AreaCustomizedSymptomList(consts.NEARBY_DISTANCE_RANGE, *loc)
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

	customized, err := s.mongoStore.AreaCustomizedSymptomList(consts.NEARBY_DISTANCE_RANGE, *loc)
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
	official, customized, err := s.findSymptomsInDB(params.Symptoms)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}
	totalSymptoms := append(official, customized...)
	symptomScore := calculateSymptomScore(totalSymptoms)
	data := schema.SymptomReportData{
		ProfileID:          account.Profile.ID.String(),
		AccountNumber:      account.Profile.AccountNumber,
		OfficialSymptoms:   official,
		Location:           schema.GeoJSON{Type: "Point", Coordinates: []float64{loc.Longitude, loc.Latitude}},
		CustomizedSymptoms: customized,
		SymptomScore:       symptomScore,
		Timestamp:          time.Now().UTC().Unix(),
	}

	err = s.mongoStore.SymptomReportSave(&data)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}
	if len(data.CustomizedSymptoms) > 0 {
		err = s.mongoStore.UpdateAreaProfileSymptom(data.CustomizedSymptoms, *loc)
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

	c.JSON(http.StatusOK, gin.H{"result": "OK"})

	return
}

func calculateSymptomScore(symptoms []schema.Symptom) float64 {
	var sum float64 = 0
	for _, symptom := range symptoms {
		sum = sum + symptom.Weight
	}
	if len(symptoms) >= 3 {
		return 2 * sum
	}
	return sum
}

func (s *Server) findSymptomsInDB(ids []string) ([]schema.Symptom, []schema.Symptom, error) {
	var syIDs []schema.SymptomType
	for _, id := range ids {
		syIDs = append(syIDs, schema.SymptomType(id))
	}
	official, customeried, _, err := s.mongoStore.IDToSymptoms(syIDs)
	return official, customeried, err
}

func (s *Server) userSymptomInfection(infectedUser *schema.SymptomReportData, loc schema.Location) error {
	err := s.mongoStore.UpdateAreaProfileSymptom(infectedUser.CustomizedSymptoms, loc)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) areaSymptomInfection(distance int, loc schema.Location) ([]schema.Symptom, error) {
	list, err := s.mongoStore.AreaCustomizedSymptomList(distance, loc)
	if err != nil {
		return nil, err
	}
	return list, nil
}
