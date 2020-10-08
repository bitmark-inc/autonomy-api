package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bitmark-inc/autonomy-api/schema"
)

// createSurvey is an api to add a survey result
func (s *Server) createSurvey(c *gin.Context) {
	var params schema.Survey

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	if params.SurveyID == "" {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, fmt.Errorf("empty survey id"))
		return
	}

	params.AccountNumber = c.GetString("requester")

	id, err := s.mongoStore.CreateSurvey(params)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": id,
	})
}
