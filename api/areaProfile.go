package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/score"
)

const (
	metricUpdateInterval = 5 * time.Minute
)

// autonomyProfile is a routing to dispatch requests between current user profile or a POI profile
func (s *Server) autonomyProfile(c *gin.Context) {
	id := c.Param("poiID")

	if id == "me" {
		s.currentAreaProfile(c)
		return
	}
}

func (s *Server) singleAreaProfile(c *gin.Context) {
	accountNumber := c.GetString("requester")

	profile, err := s.mongoStore.GetProfile(accountNumber)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	poiID, err := primitive.ObjectIDFromHex(c.Param("poiID"))
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("invalid POI ID"))
		return
	}

	metric, err := s.mongoStore.SyncAccountPOIMetrics(accountNumber, profile.ScoreCoefficient, poiID)
	if err != nil {
		c.Error(err)
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, metric)
}

func (s *Server) currentAreaProfile(c *gin.Context) {
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

	individualMetric := profile.IndividualMetric
	metric := profile.Metric

	if profile.Location != nil {
		location := schema.Location{
			Latitude:  profile.Location.Coordinates[1],
			Longitude: profile.Location.Coordinates[0],
		}

		metricLastUpdate := time.Unix(metric.LastUpdate, 0)
		var coefficient *schema.ScoreCoefficient

		if time.Since(metricLastUpdate) >= metricUpdateInterval {
			// will sync with coefficient = nil
		} else if coefficient = profile.ScoreCoefficient; coefficient != nil && coefficient.UpdatedAt.Sub(metricLastUpdate) > 0 {
			// will sync with coefficient = profile.ScoreCoefficient
		} else {
			autonomyScore, autonomyScoreDelta := score.CalculateIndividualAutonomyScore(individualMetric, metric)
			c.JSON(http.StatusOK, gin.H{
				"autonomy_score":       autonomyScore,
				"autonomy_score_delta": autonomyScoreDelta,
				"individual":           individualMetric,
				"neighbor":             metric,
			})
			return
		}

		m, err := s.mongoStore.SyncAccountMetrics(account.AccountNumber, coefficient, location)
		if err != nil {
			c.Error(err)
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		} else {
			metric = *m
		}
	}

	autonomyScore, autonomyScoreDelta := score.CalculateIndividualAutonomyScore(individualMetric, metric)
	c.JSON(http.StatusOK, gin.H{
		"autonomy_score":       autonomyScore,
		"autonomy_score_delta": autonomyScoreDelta,
		"individual":           individualMetric,
		"neighbor":             metric,
	})
}
