package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bitmark-inc/autonomy-api/schema"
)

func (s *Server) recordConsent(c *gin.Context) {
	var r schema.ConsentRecord
	if err := c.BindJSON(&r); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	r.Timestamp = time.Now()

	if err := s.mongoStore.RecordConsent(r); err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}
