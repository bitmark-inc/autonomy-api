package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bitmark-inc/autonomy-api/schema"
)

func (s *Server) createFeedback(c *gin.Context) {
	var params schema.Feedback

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	params.AccountNumber = c.GetString("requester")

	id, err := s.mongoStore.CreateFeedback(params)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": id,
	})
}
