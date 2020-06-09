package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bitmark-inc/autonomy-api/consts"
	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/score"
)

func (s *Server) getSymptomMetrics(c *gin.Context) {
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

	profileID := account.ProfileID.String()

	now := time.Now().UTC()

	meToday, meYesterday, err := s.mongoStore.GetSymptomCount(profileID, nil, 0, now)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}
	communityToday, communityYesterday, err := s.mongoStore.GetSymptomCount("", loc, consts.NEARBY_DISTANCE_RANGE, now)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}
	reporterCount, _, err := s.mongoStore.GetNearbyReportingUserCount(schema.ReportTypeSymptom, consts.NEARBY_DISTANCE_RANGE, *loc, now)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"me": gin.H{
			"total_today": meToday,
			"delta":       score.ChangeRate(float64(meToday), float64(meYesterday)),
		},
		"community": gin.H{
			"avg_today": score.DivOrDefault(float64(communityToday), float64(reporterCount), 0.0),
			"delta":     score.ChangeRate(float64(communityToday), float64(communityYesterday)),
		},
	})
}

func (s *Server) getBehaviorMetrics(c *gin.Context) {
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

	profileID := account.ProfileID.String()

	now := time.Now().UTC()

	meToday, meYesterday, err := s.mongoStore.GetBehaviorCount(profileID, nil, 0, now)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}
	communityToday, communityYesterday, err := s.mongoStore.GetBehaviorCount("", loc, consts.NEARBY_DISTANCE_RANGE, now)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}
	reporterCount, _, err := s.mongoStore.GetNearbyReportingUserCount(schema.ReportTypeBehavior, consts.NEARBY_DISTANCE_RANGE, *loc, now)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"me": gin.H{
			"total_today": meToday,
			"delta":       score.ChangeRate(float64(meToday), float64(meYesterday)),
		},
		"community": gin.H{
			"avg_today": score.DivOrDefault(float64(communityToday), float64(reporterCount), 0.0),
			"delta":     score.ChangeRate(float64(communityToday), float64(communityYesterday)),
		},
	})
}
