package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bitmark-inc/autonomy-api/geo"
	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/score"
)

func (s *Server) calculateScore(c *gin.Context) {
	var params struct {
		Places []struct {
			Address string `json:"address"`
		} `json:"places"`
	}

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	scores := make([]*float64, 0, len(params.Places))

	for _, p := range params.Places {
		if p.Address == "" {
			scores = append(scores, nil)
			continue
		}

		lat, lon, err := geo.LookupCoordinate(p.Address)
		if err != nil {
			if err == geo.ErrLocationNotFound {
				scores = append(scores, nil)
				continue
			}
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}

		rawMetrics, err := s.mongoStore.CollectRawMetrics(schema.Location{Latitude: lat, Longitude: lon})
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}

		metric := score.CalculateMetric(*rawMetrics, nil)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
			return
		}

		scores = append(scores, &metric.Score)
	}

	c.JSON(http.StatusOK, gin.H{"results": scores})
}
