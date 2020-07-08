package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bitmark-inc/autonomy-api/schema"
)

// accountRegister is the API for register a new account
func (s *Server) accountRegister(c *gin.Context) {
	accountNumber := c.GetString("requester")

	var params struct {
		EncPubKey string                 `json:"enc_pub_key" binding:"required"`
		Metadata  map[string]interface{} `json:"metadata"`
	}

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	err := s.mongoStore.CreateAccount(accountNumber, params.EncPubKey, params.Metadata)
	if err != nil {
		abortWithEncoding(c, http.StatusForbidden, errorAccountTaken, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": "OK",
	})
}

// accountDetail is the API to query an account
func (s *Server) accountDetail(c *gin.Context) {
	a := c.MustGet("account")
	profile, ok := a.(*schema.Profile)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": map[string]interface{}{
			"id":             profile.ID,
			"account_number": profile.AccountNumber,
			"metadata":       profile.Metadata,
		},
	})
}

// accountUpdateMetadata is the API to update metadata for a user
func (s *Server) accountUpdateMetadata(c *gin.Context) {
	accountNumber := c.GetString("requester")

	var params struct {
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := c.BindJSON(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorCannotParseRequest, err)
		return
	}

	if err := s.mongoStore.UpdateAccountMetadata(accountNumber, params.Metadata); err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}

// accountDelete is the API to remove an account from our service
func (s *Server) accountDelete(c *gin.Context) {
	accountNumber := c.GetString("requester")

	if err := s.mongoStore.DeleteAccount(accountNumber); err != nil {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}

// accountHere is an api to acking for an account
func (s *Server) accountHere(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
