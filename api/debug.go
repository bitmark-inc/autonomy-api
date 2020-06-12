package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bitmark-inc/autonomy-api/consts"
	"github.com/bitmark-inc/autonomy-api/schema"
)

func (s *Server) currentAreaDebugData(c *gin.Context) {
	account, ok := c.MustGet("account").(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	profile, err := s.mongoStore.GetProfile(account.AccountNumber)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	metric := profile.Metric
	var userCount, aqiNumber, symptomCount int

	if profile.Location != nil {
		loc := schema.Location{
			Latitude:  profile.Location.Coordinates[1],
			Longitude: profile.Location.Coordinates[0],
		}
		metricLastUpdate := time.Unix(metric.LastUpdate, 0)
		if time.Since(metricLastUpdate) >= metricUpdateInterval {
			m, err :=
				s.mongoStore.SyncAccountMetrics(account.AccountNumber, profile.ScoreCoefficient, loc)
			if err != nil {
				c.Error(err)
				abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
				return
			} else {
				metric = *m
			}
		}

		aqiNumber, err = s.aqiClient.Get(loc.Latitude, loc.Longitude)
		if err != nil {
			c.Error(err)
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}

		nearAccounts, err := s.mongoStore.NearestDistance(consts.NEARBY_DISTANCE_RANGE, loc)
		if err != nil {
			c.Error(err)
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		userCount = len(nearAccounts)

		now := time.Now().UTC()
		todayStartAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		todayStartAtUnix := todayStartAt.Unix()
		tomorrowStartAtUnix := todayStartAt.AddDate(0, 0, 1).Unix()
		distribution, err := s.mongoStore.FindSymptomDistribution("", &loc, consts.NEARBY_DISTANCE_RANGE, todayStartAtUnix, tomorrowStartAtUnix, true)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		for _, cnt := range distribution {
			symptomCount += cnt
		}
	}

	debug := schema.Debug{
		Metrics:  metric,
		Users:    userCount,
		AQI:      aqiNumber,
		Symptoms: symptomCount,
	}

	c.JSON(http.StatusOK, debug)
}

func (s *Server) poiDebugData(c *gin.Context) {
	poiID, err := primitive.ObjectIDFromHex(c.Param("poiID"))
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("invalid POI ID"))
		return
	}

	poi, err := s.mongoStore.GetPOI(poiID)
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("error getting poi"))
		return
	}

	metric := poi.Metric
	var userCount, aqiNumber, symptomCount int

	if poi.Location != nil {
		loc := schema.Location{
			Latitude:  poi.Location.Coordinates[1],
			Longitude: poi.Location.Coordinates[0],
			AddressComponent: schema.AddressComponent{
				Country: poi.Country,
				State:   poi.State,
				County:  poi.County,
			},
		}
		metricLastUpdate := time.Unix(metric.LastUpdate, 0)
		if time.Since(metricLastUpdate) >= metricUpdateInterval {
			m, err :=
				s.mongoStore.SyncPOIMetrics(poiID, loc)
			if err != nil {
				c.Error(err)
				abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
				return
			} else {
				metric = *m
			}
		}

		aqiNumber, err = s.aqiClient.Get(loc.Latitude, loc.Longitude)
		if err != nil {
			c.Error(err)
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}

		nearAccounts, err := s.mongoStore.NearestDistance(consts.NEARBY_DISTANCE_RANGE, loc)
		if err != nil {
			c.Error(err)
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		userCount = len(nearAccounts)

		now := time.Now().UTC()
		todayStartAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		todayStartAtUnix := todayStartAt.Unix()
		tomorrowStartAtUnix := todayStartAt.AddDate(0, 0, 1).Unix()
		distribution, err := s.mongoStore.FindSymptomDistribution("", &loc, consts.NEARBY_DISTANCE_RANGE, todayStartAtUnix, tomorrowStartAtUnix, true)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		for _, cnt := range distribution {
			symptomCount += cnt
		}

	}

	debug := schema.Debug{
		Metrics:  metric,
		Users:    userCount,
		AQI:      aqiNumber,
		Symptoms: symptomCount,
	}

	c.JSON(http.StatusOK, debug)
}
