package api

import (
	"net/http"

	"github.com/bitmark-inc/autonomy-api/store"
	"github.com/gin-gonic/gin"
)

// askForHelp is the API for asking help from others
func (s *Server) askForHelp(c *gin.Context) {
	requester := c.GetString("requester")

	var params struct {
		Subject      string `json:"subject"`
		Needs        string `json:"exact_needs"`
		MeetingPlace string `json:"meeting_location"`
		ContactInfo  string `json:"contact_info"`
	}

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	req, err := s.store.RequestHelp(requester, params.Subject, params.Needs, params.MeetingPlace, params.ContactInfo)
	if err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	// TODO: broadcast a notification to surrounding users
	c.JSON(http.StatusOK, req)
	return
}

// queryHelps is the API for return a list of helps if a help id is not given.
// It returns a specific help request if the help id is provided.
func (s *Server) queryHelps(c *gin.Context) {
	var result gin.H
	helpID := c.Param("helpID")

	if helpID != "" {
		help, err := s.store.GetHelp(helpID)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		result = gin.H{
			"result": help,
		}
	} else {

	}

	c.JSON(http.StatusOK, result)
	return
}

// answerHelp is the API for answer a help
func (s *Server) answerHelp(c *gin.Context) {
	id := c.Param("helpID")
	requester := c.GetString("requester")

	if err := s.store.AnswerHelp(requester, id); err != nil {
		if err == store.ErrRequestNotExist {
			abortWithEncoding(c, http.StatusNotFound, errorRequestNotExist, err)
		} else {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}
