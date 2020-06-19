package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bitmark-inc/autonomy-api/schema"
)

// listResources is an API handler for resources listing
func (s *Server) listResources(c *gin.Context) {
	account, ok := c.MustGet("account").(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	var params struct {
		Suggestion bool   `form:"suggestion"`
		Language   string `form:"lang"`
	}

	if err := c.Bind(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	lang := "en"
	if params.Language != "" {
		lang = params.Language
	}

	if !params.Suggestion {
		abortWithEncoding(c, http.StatusInternalServerError, errorResourceNotSupport)
		return
	}

	resources, err := s.mongoStore.ListSuggestedResources(account.Profile.ID.String(), lang)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}
	c.JSON(200, gin.H{"resources": resources})
}
