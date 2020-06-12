package api

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/store"
)

func (s *Server) updatePOIRating(c *gin.Context) {
	account, ok := c.MustGet("account").(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	poiID, err := primitive.ObjectIDFromHex(c.Param("poiID"))
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("invalid POI ID"))
		return
	}

	var params struct {
		Language string `form:"lang"`
	}

	if err := c.BindQuery(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}
	if "" == params.Language {
		params.Language = "en"
	}
	type userRating struct {
		ResourceID string `json:"resource_id"`
		Score      int    `json:"score"`
	}

	var body struct {
		Ratings []userRating `json:"ratings"`
	}

	if err := c.BindJSON(&body); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	var profileMetric schema.ProfileRatingsMetric

	for _, r := range body.Ratings {
		name, resovErr := store.ResolveResourceNameByID(r.ResourceID, params.Language)
		if resovErr != nil || "" == name {
			c.Error(fmt.Errorf("resovError:%v", resovErr))
			continue
		}
		rating := schema.RatingResource{
			Resource: schema.Resource{ID: r.ResourceID, Name: name},
			Score:    float64(r.Score),
		}
		profileMetric.Resources = append(profileMetric.Resources, rating)
	}

	profileMetric.LastUpdate = time.Now().Unix()
	err = s.mongoStore.UpdateProfilePOIRatingMetric(account.AccountNumber, poiID, profileMetric)
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorProfileNotUpdate, fmt.Errorf("UpdateProfilePOIRatingMetric update rating metric error"))
		return
	}

	if err := s.mongoStore.UpdatePOIRatingMetric(poiID, profileMetric.Resources); err != nil {
		switch err {
		case store.ErrPOINotFound:
			abortWithEncoding(c, http.StatusBadRequest, errorUnknownPOI)
		default:
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (s *Server) getProfileRatings(c *gin.Context) {
	account, ok := c.MustGet("account").(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}
	poiID := c.Param("poiID")
	metric, err := s.mongoStore.GetProfilePOIRatingMetric(account.AccountNumber, poiID)

	if err != nil {
		c.Error(err)
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}
	var params struct {
		Language string `form:"lang"`
	}

	if err := c.BindQuery(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}
	if "" == params.Language {
		params.Language = "en"
	}

	for i, r := range metric.Resources {
		name, resovErr := store.ResolveResourceNameByID(r.ID, params.Language)
		if resovErr != nil || "" == name {
			c.Error(fmt.Errorf("resoveResourceNameByIDError:%v", resovErr))
			continue
		}
		metric.Resources[i].Name = name
	}

	sort.SliceStable(metric.Resources, func(i, j int) bool {
		return metric.Resources[i].Score > metric.Resources[j].Score // Inverse sort
	})
	c.JSON(http.StatusOK, gin.H{"ratings": metric.Resources})
}
