package api

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/store"
)

func (s *Server) setPOIRating(c *gin.Context) {
	poiID := c.Param("poiID")
	macaroonTokenPDS := c.GetHeader("X-FORWARD-MACAROON-PDS")
	macaroonTokenCDS := c.GetHeader("X-FORWARD-MACAROON-CDS")
	if macaroonTokenPDS == "" || macaroonTokenCDS == "" {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, fmt.Errorf("invalid maracoon token"))
		return
	}

	var params struct {
		Ratings map[string]float64 `json:"ratings"`
	}

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	err := s.dataStore.SetPOIRating(macaroonTokenPDS, poiID, params.Ratings)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	err = s.dataStore.SetPOICommunityRating(macaroonTokenCDS, poiID, params.Ratings)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "ok"})
}

func (s *Server) getPOIRating(c *gin.Context) {
	poiID := c.Param("poiID")
	macaroonToken := c.GetHeader("X-FORWARD-MACAROON-PDS")
	if macaroonToken == "" {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, fmt.Errorf("invalid maracoon token"))
		return
	}

	rating, err := s.dataStore.GetPOIRating(macaroonToken, poiID)
	if err != nil {
		fmt.Println(err.Error())
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	if rating.Ratings == nil {
		rating.Ratings = map[string]float64{}
	}

	c.JSON(http.StatusOK, rating)
}

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

	type userRating struct {
		Resource schema.Resource `json:"resource"`
		Score    int             `json:"score"`
	}

	var body struct {
		Ratings []userRating `json:"ratings"`
	}

	if err := c.BindJSON(&body); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	poiResources, err := s.mongoStore.GetPOIResourceMetric(poiID)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}
	resourceNames := make(map[string]string)
	for _, r := range poiResources.Resources {
		resourceNames[r.ID] = r.Name
	}

	var profileMetric schema.ProfileRatingsMetric
	for _, r := range body.Ratings {
		name := resourceNames[r.Resource.ID]

		if r.Score > 0 { // score zero means unrated, score cant be zero
			rating := schema.RatingResource{
				Resource: schema.Resource{ID: r.Resource.ID, Name: name},
				Score:    float64(r.Score),
			}
			profileMetric.Resources = append(profileMetric.Resources, rating)
		}

	}

	if err := s.mongoStore.UpdatePOIRatingMetric(account.AccountNumber, poiID, profileMetric.Resources); err != nil {
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
		c.Error(fmt.Errorf("Can not get account number"))
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}
	poiID := c.Param("poiID")

	poiObj, err := primitive.ObjectIDFromHex(poiID)
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("invalid POI ID"))
		return
	}
	metric, err := s.mongoStore.GetProfilePOIRatingMetric(account.AccountNumber, poiID)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	poiMetric, getPoiErr := s.mongoStore.GetPOIResourceMetric(poiObj)

	if err != nil {
		c.Error(fmt.Errorf("GetPOIResourceMetric:%v", getPoiErr))
		c.JSON(http.StatusOK, gin.H{"ratings": metric.Resources})
		return
	}

	profileMap := make(map[string]schema.RatingResource)
	for _, v := range metric.Resources { // make a current resources map
		profileMap[v.ID] = v
	}

	for _, r := range poiMetric.Resources {
		_, ok := profileMap[r.ID]
		if !ok {
			notInProfile := schema.RatingResource{
				Resource: r.Resource,
				Score:    0,
			}
			metric.Resources = append(metric.Resources, notInProfile)
		}
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

	poiResources, err := s.mongoStore.GetPOIResourceMetric(poiObj)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}
	resourceNames := make(map[string]string)
	for _, r := range poiResources.Resources {
		resourceNames[r.ID] = r.Name
	}

	for i, r := range metric.Resources {
		name, _ := store.ResolveResourceNameByID(r.ID, params.Language)
		if "" == name { // show original name
			name = resourceNames[r.ID]
		}
		metric.Resources[i].Name = name
	}

	sort.SliceStable(metric.Resources, func(i, j int) bool {
		return metric.Resources[i].Score > metric.Resources[j].Score // Inverse sort
	})
	if nil == metric.Resources {
		metric.Resources = []schema.RatingResource{}
	}
	c.JSON(http.StatusOK, gin.H{"ratings": metric.Resources})
}
