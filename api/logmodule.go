package api

import (
	"net/http/httputil"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// DumpRequest is a middleware to dump incoming http requests if the
// trade mode is enabled.
func (s *Server) DumpRequest(c *gin.Context) {
	if s.traceMode {
		dump, err := httputil.DumpRequest(c.Request, false)
		if err != nil {
			log.WithFields(logrus.Fields{
				"prefix":  "gin",
				"status":  c.Writer.Status(),
				"method":  c.Request.Method,
				"headers": c.Request.Header.Get("Authorization"),
				"path":    c.Request.URL.Path,
			}).Error("fail to dump request")
		}

		log.WithFields(logrus.Fields{
			"prefix": "gin",
			"req":    string(dump),
		}).Debug("incoming request")
	}

	c.Next()
}
