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
)

func (s *Server) createBehavior(c *gin.Context) {
	var params schema.Behavior

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	if params.Name == "" {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters)
		return
	}

	id, err := s.mongoStore.CreateBehavior(params)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": id,
	})
	return
}

func (s *Server) goodBehaviors(c *gin.Context) {
	a := c.MustGet("account")
	account, ok := a.(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

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

	var loc *schema.Location
	loc = account.Profile.State.LastLocation
	if nil == loc {
		abortWithEncoding(c, http.StatusBadRequest, errorUnknownAccountLocation)
		return
	}

	behaviors, err := s.mongoStore.ListOfficialBehavior(lang)
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorUnknownAccountLocation)
		return
	}

	nonOfficialBehaviors, err := s.mongoStore.FindNearbyNonOfficialBehaviors(consts.NEARBY_DISTANCE_RANGE, *loc)
	if err != nil {
		c.Error(err)
	}

	if nonOfficialBehaviors != nil {
		behaviors = append(behaviors, nonOfficialBehaviors...)
	}

	c.JSON(http.StatusOK, gin.H{"behaviors": behaviors})
}

func (s *Server) getBehaviorsV2(c *gin.Context) {
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

	official, err := s.mongoStore.ListOfficialBehavior(lang)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	if params.All {
		customized, err := s.mongoStore.ListCustomizedBehaviors()
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"official_behaviors":   official,
			"customized_behaviors": customized,
		})
		return
	}

	customized, err := s.mongoStore.FindNearbyNonOfficialBehaviors(consts.NEARBY_DISTANCE_RANGE, *loc)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"official_behaviors":     official,
		"neighborhood_behaviors": customized,
	})
}

func (s *Server) reportBehaviors(c *gin.Context) {
	a := c.MustGet("account")
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

	var params struct {
		Behaviors []string `json:"behaviors"`
	}

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}
	if 0 == len(params.Behaviors) {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, errors.New(errorMessageMap[1010]))
		return
	}

	behaviors, err := s.mongoStore.FindBehaviorsByIDs(params.Behaviors)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	data := schema.BehaviorReportData{
		ProfileID:     account.Profile.ID.String(),
		AccountNumber: account.Profile.AccountNumber,
		Behaviors:     behaviors,
		Location:      schema.GeoJSON{Type: "Point", Coordinates: []float64{loc.Longitude, loc.Latitude}},
		Timestamp:     time.Now().UTC().Unix(),
	}

	err = s.mongoStore.GoodBehaviorSave(&data)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	if _, err := s.mongoStore.SyncProfileIndividualMetrics(account.Profile.ID.String()); err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	_, nonOfficial := schema.SplitBehaviors(behaviors)
	if len(nonOfficial) > 0 {
		err = s.mongoStore.UpdateAreaProfileBehavior(nonOfficial, *loc)
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
	if err != nil {
		c.Error(err)
	}
	if nil == err {
		go func() {
			if err := utils.TriggerPOIUpdate(*s.cadenceClient, c, pois); err != nil {
				sentry.CaptureException(err)
			}
		}()
	} else {
		c.Error(err)
	}
	c.JSON(http.StatusOK, gin.H{"result": "OK"})
	return
}
