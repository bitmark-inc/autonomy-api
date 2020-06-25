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

// autonomyProfile is a handler that dispatches requests between current user profile or a POI profile
// There are two type for POI profile. One is with a POI ID. Another is to provide a coordinates.
func (s *Server) autonomyProfile(c *gin.Context) {
	var params struct {
		Me           bool    `form:"me"`
		POIID        string  `form:"poi_id"`
		Latitude     float64 `form:"lat"`
		Longitude    float64 `form:"lng"`
		Language     string  `form:"lang"`
		AllResources bool    `form:"all_resources"`
	}

	if err := c.Bind(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	if "" == params.Language {
		params.Language = "en"
	}

	c.Set("language", params.Language)
	c.Set("allResources", params.AllResources)

	if params.Me {
		s.currentAreaProfile(c)
		return
	} else if params.POIID != "" {
		s.placeProfile(c, params.POIID, nil)
		return
	} else if params.Latitude != 0 && params.Longitude != 0 {
		s.placeProfile(c, "", &schema.Location{
			Longitude: params.Longitude,
			Latitude:  params.Latitude,
		})
		return
	} else {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters)
		return
	}
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

// placeProfileResponse is a struct for response of the autonomy profile of a given POI
type placeProfileResponse struct {
	ID              string                     `json:"id"`
	Alias           string                     `json:"alias"`
	Address         string                     `json:"address"`
	Owned           bool                       `json:"owned"`
	Rating          bool                       `json:"rating"`
	HasMoreResource bool                       `json:"has_more_resources"`
	Metric          schema.Metric              `json:"neighbor"`
	Resources       []schema.POIResourceRating `json:"resources"`
	Score           float64                    `json:"autonomy_score"`
	ScoreDelta      float64                    `json:"autonomy_score_delta"`
}

// summarizePlaceProfile summarize profile response for a given POI. It takes profile and language
// into consideration to generate proper response.
func summarizePlaceProfile(poi *schema.POI, profile *schema.Profile, language string, allResources bool) interface{} {
	var resp placeProfileResponse

	var profilePOI *schema.ProfilePOI
	for _, p := range profile.PointsOfInterest {
		if p.ID == poi.ID {
			profilePOI = &p
			break
		}
	}

	resources := poi.ResourceRatings.Resources
	if len(resources) == 0 {
		resources = []schema.POIResourceRating{}
	} else {
		sort.SliceStable(resources, func(i, j int) bool {
			return resources[i].Ratings > resources[j].Ratings // Inverse sort
		})

		if !allResources { // return 10 records and indicate more or not
			if len(resources) > 10 {
				resources = resources[:10]
				resp.HasMoreResource = true
			}
		}

		for i, r := range resources {
			name, _ := store.ResolveResourceNameByID(r.ID, language)
			if "" == name { // show original name
				name = r.Name
			}
			resources[i].Name = name
		}
	}

	resp.ID = poi.ID.Hex()
	resp.Alias = poi.Alias
	resp.Address = poi.Address

	if profilePOI != nil && profilePOI.Monitored {
		resp.Alias = profilePOI.Alias
		resp.Address = profilePOI.Address
		resp.Owned = true
	}

	if len(resources) > 0 {
		resp.Rating = true
	}
	resp.Score = poi.Score
	resp.ScoreDelta = poi.ScoreDelta
	resp.Metric = poi.Metric
	resp.Resources = resources

	return resp
}

func (s *Server) placeProfile(c *gin.Context, id string, location *schema.Location) {
	accountNumber := c.GetString("requester")

	profile, err := s.mongoStore.GetProfile(accountNumber)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	allResources := c.GetBool("allResources")
	language := c.GetString("language")
	resources := []schema.POIResourceRating{}

	if id != "" {
		poiID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("invalid POI ID"))
			return
		}

		// Get POI resource
		poi, err := s.mongoStore.GetPOI(poiID)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}

		resp := summarizePlaceProfile(poi, profile, language, allResources)
		c.JSON(http.StatusOK, resp)
	} else if location != nil {
		// Get POI resource by coordinates
		poi, err := s.mongoStore.GetPOIByCoordinates(*location)
		if err != nil {
			if err != store.ErrPOINotFound {
				abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
				return
			}
		}

		if poi != nil {
			resp := summarizePlaceProfile(poi, profile, language, allResources)
			c.JSON(http.StatusOK, resp)
		} else {
			// Collect profile by location if there is no poi meet the location
			metric, err := s.mongoStore.CollectRawMetrics(*location)
			if err != nil {
				abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
				return
			}
			*metric = score.CalculateMetric(*metric, nil)
			score, _, scoreDelta := score.CalculatePOIAutonomyScore(nil, *metric)

			resp := placeProfileResponse{
				Score:      score,
				ScoreDelta: scoreDelta,
				Metric:     *metric,
				Resources:  resources,
			}
			c.JSON(http.StatusOK, resp)
		}
	} else {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters)
		return
	}
}
