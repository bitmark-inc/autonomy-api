package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/store"
	"github.com/bitmark-inc/autonomy-api/utils"
)

type poiRequestBody struct {
	ID       string           `json:"poi_id"`
	Alias    string           `json:"alias"`
	Address  string           `json:"address"`
	Location *schema.Location `json:"location"`
	Score    float64          `json:"score"`
	Types    []string         `json:"types,omitempty"`
}

func (s *Server) addPOI(c *gin.Context) {
	var req poiRequestBody
	if err := c.BindJSON(&req); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	if req.Location == nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("location not provided"))
		return
	}

	placeType := utils.ReadPlaceType(req.Types)

	poi, err := s.mongoStore.AddPOI(req.Alias, req.Address, placeType, req.Location.Longitude, req.Location.Latitude)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, schema.POIDetail{
		ProfilePOI: schema.ProfilePOI{
			ID:      poi.ID,
			Alias:   poi.Alias,
			Address: poi.Address,
			Score:   poi.Metric.Score,
		},
		Location: req.Location,
	})
}

func (s *Server) addOwnPOI(c *gin.Context) {
	accountNumber := c.GetString("requester")

	var req poiRequestBody
	if err := c.BindJSON(&req); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	poiID, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("invalid POI ID"))
		return
	}

	poi, err := s.mongoStore.GetPOI(poiID)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	poiDesc := schema.ProfilePOI{
		ID:        poi.ID,
		Alias:     poi.Alias,
		Address:   poi.Address,
		PlaceType: poi.PlaceType,
		Monitored: true,
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.mongoStore.AppendPOIToAccountProfile(accountNumber, poiDesc); err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	resp := schema.POIDetail{
		ProfilePOI: poiDesc,
		Location: &schema.Location{
			Longitude: poi.Location.Coordinates[0],
			Latitude:  poi.Location.Coordinates[1],
		},
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) listOwnPOI(c *gin.Context) {
	account, ok := c.MustGet("account").(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}
	pois, err := s.mongoStore.ListPOI(account.AccountNumber)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, pois)
	return
}

func (s *Server) listPOI(c *gin.Context) {
	account, ok := c.MustGet("account").(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	var params struct {
		ResourceID string `form:"resource_id"`
	}

	if err := c.Bind(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	if params.ResourceID != "" {
		location := account.Profile.State.LastLocation
		if nil == location {
			abortWithEncoding(c, http.StatusBadRequest, errorUnknownAccountLocation)
			return
		}

		pois, err := s.mongoStore.ListPOIByResource(params.ResourceID, *location)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}

		response := make([]schema.POIDetail, len(pois))

		for i, p := range pois {
			response[i] = schema.POIDetail{
				ProfilePOI: schema.ProfilePOI{
					ID:        p.ID,
					Address:   p.Address,
					Alias:     p.Alias,
					Score:     p.Score,
					PlaceType: p.PlaceType,
				},
				Location: &schema.Location{
					Longitude: p.Location.Coordinates[0],
					Latitude:  p.Location.Coordinates[1],
				},
				Distance:      p.Distance,
				ResourceScore: p.ResourceScore,
			}
		}

		c.JSON(http.StatusOK, response)
		return
	} else {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters)
		return
	}
}

func (s *Server) updatePOIAlias(c *gin.Context) {
	account, ok := c.MustGet("account").(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	var body poiRequestBody
	if err := c.BindJSON(&body); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	poiID, err := primitive.ObjectIDFromHex(c.Param("poiID"))
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("invalid POI ID"))
		return
	}

	if err := s.mongoStore.UpdatePOIAlias(account.AccountNumber, body.Alias, poiID); err != nil {
		switch err {
		case store.ErrPOINotFound:
			abortWithEncoding(c, http.StatusBadRequest, errorUnknownPOI)
		default:
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}

func (s *Server) updatePOIOrder(c *gin.Context) {

	account, ok := c.MustGet("account").(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	var params struct {
		Order []string `json:"order"`
	}

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	if err := s.mongoStore.UpdatePOIOrder(account.AccountNumber, params.Order); err != nil {
		switch err {
		case store.ErrPOIListNotFound:
			abortWithEncoding(c, http.StatusInternalServerError, errorPOIListNotFound, err)
			return
		case store.ErrPOIListMismatch:
			abortWithEncoding(c, http.StatusInternalServerError, errorPOIListMissmatch, err)
			return
		default:
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}

func (s *Server) deletePOI(c *gin.Context) {
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

	if err := s.mongoStore.DeletePOI(account.AccountNumber, poiID); err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}

// addPOIResources add resources into a POI
func (s *Server) addPOIResources(c *gin.Context) {
	poiID, err := primitive.ObjectIDFromHex(c.Param("poiID"))
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("invalid POI ID"))
		return
	}

	var query struct {
		Language string `form:"lang"`
	}

	if err := c.BindQuery(&query); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	var params struct {
		ResourceIDs      []string `json:"resource_ids"`
		NewResourceNames []string `json:"new_resource_names"`
	}
	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	addedResources := make([]schema.Resource, 0, len(params.ResourceIDs)+len(params.NewResourceNames))
	for _, id := range params.ResourceIDs {
		addedResources = append(addedResources, schema.Resource{ID: id})
	}
	for _, name := range params.NewResourceNames {
		if name == "" {
			abortWithEncoding(c, http.StatusBadRequest, errorEmptyPOIResourceName)
			return
		}
		addedResources = append(addedResources, schema.Resource{Name: name})
	}

	resources, err := s.mongoStore.AddPOIResources(poiID, addedResources, query.Language)
	if err != nil {
		switch err {
		case store.ErrPOINotFound:
			abortWithEncoding(c, http.StatusBadRequest, errorUnknownPOI)
		default:
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"resources": resources,
	})
	return
}

// getPOIResources add resources from a POI
func (s *Server) getPOIResources(c *gin.Context) {
	poiID, err := primitive.ObjectIDFromHex(c.Param("poiID"))
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("invalid POI ID"))
		return
	}

	var params struct {
		Language      string `form:"lang"`
		ImportantOnly bool   `form:"important"`
	}

	if err := c.Bind(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	resources, err := s.mongoStore.GetPOIResources(poiID, params.ImportantOnly, params.Language)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"resources": resources,
	})
	return
}
