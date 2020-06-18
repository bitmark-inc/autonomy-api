package api

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/score"
	"github.com/bitmark-inc/autonomy-api/store"
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
	s.placeProfile(c)
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

	if time.Since(time.Unix(individualMetric.LastUpdate, 0)) >= metricUpdateInterval {
		i, err := s.mongoStore.SyncProfileIndividualMetrics(profile.ID)
		if err != nil {
			c.Error(err)
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		} else {
			individualMetric = *i
		}
	}

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

func (s *Server) placeProfile(c *gin.Context) {
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

	// resources
	var params struct {
		Language     string `form:"lang"`
		AllResources bool   `form:"all_resources"`
	}
	if err := c.Bind(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	if "" == params.Language {
		params.Language = "en"
	}

	var profilePoi schema.ProfilePOI
	poiFound := false
	for _, p := range profile.PointsOfInterest {
		if p.ID == poiID {
			profilePoi = p
			poiFound = true
		}
	}
	if !poiFound {
		abortWithEncoding(c, http.StatusBadRequest, errorNoPOIInProfile)
	}

	// Get POI Resources
	poi, err := s.mongoStore.GetPOI(poiID)
	if err != nil {
		c.Error(err)
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}
	var resp struct {
		ID      string `json:"id"`
		Alias   string `json:"alias"`
		Address string `json:"address"`
		//Location        *schema.Location           `json:"location"`
		Rating          bool                       `json:"rating"`
		HasMoreResource bool                       `json:"has_more_resources"`
		Metric          schema.Metric              `json:"neighbor"`
		Resources       []schema.POIResourceRating `json:"resources"`
		Score           float64                    `json:"autonomy_score"`
		ScoreDelta      float64                    `json:"autonomy_score_delta"`
	}

	resources := poi.ResourceRatings.Resources
	sort.SliceStable(resources, func(i, j int) bool {
		return resources[i].Ratings > resources[j].Ratings // Inverse sort
	})

	if !params.AllResources { // return 10 records and indicate more or not
		if len(resources) > 10 {
			resources = resources[:10]
			resp.HasMoreResource = true
		}
	}

	for i, r := range resources {
		name, _ := store.ResolveResourceNameByID(r.ID, params.Language)
		if "" == name { // show original name
			name = r.Name
		}
		resources[i].Name = name
	}

	resp.ID = poi.ID.Hex()
	resp.Alias = profilePoi.Alias
	resp.Address = profilePoi.Address

	if len(profilePoi.ResourceRatings.Resources) > 0 {
		resp.Rating = true
	}
	resp.Score = poi.Score
	resp.ScoreDelta = poi.ScoreDelta
	resp.Metric = poi.Metric
	resp.Resources = resources

	c.JSON(http.StatusOK, resp)
}
