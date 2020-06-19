package api

import "github.com/bitmark-inc/autonomy-api/store"

var (
	errorMessageMap = map[int64]string{
		999:  "internal server error",
		1000: "invalid signature",
		1001: "invalid authorization format",
		1002: "difference between the request time and the current time is too large",
		1003: "invalid token",

		1006: "invalid value of client version",
		1007: "API for this client version has been discontinued",
		1008: "resource is not supported",

		1010: "invalid parameters",
		1011: "cannot parse request",

		1100: "his account has been registered or has been taken",
		1101: "account not found",
		1102: "the account is under deletion",
		1103: "query score error",
		1104: "unknown account location",
		1105: "update score error",
		1106: "unknown POI",
		1107: "no POI in the profile",

		1200: store.ErrRequestNotExist.Error(),
		1201: store.ErrMultipleRequestMade.Error(),

		1300: store.ErrPOIListNotFound.Error(),
		1301: store.ErrPOIListMismatch.Error(),
		1302: store.ErrEmptyPOIResourceName.Error(),
	}

	errorInternalServer             = errorJSON(999)
	errorInvalidSignature           = errorJSON(1000)
	errorInvalidAuthorizationFormat = errorJSON(1001)
	errorRequestTimeTooSkewed       = errorJSON(1002)
	errorInvalidToken               = errorJSON(1003)
	errorInvalidClientVersion       = errorJSON(1006)
	errorUnsupportedClientVersion   = errorJSON(1007)
	errorResourceNotSupport         = errorJSON(1008)

	errorInvalidParameters  = errorJSON(1010)
	errorCannotParseRequest = errorJSON(1011)

	errorAccountTaken           = errorJSON(1100)
	errorAccountNotFound        = errorJSON(1101)
	errorAccountDeleting        = errorJSON(1102)
	errorScore                  = errorJSON(1103)
	errorUnknownAccountLocation = errorJSON(1104)
	errorUpdateScore            = errorJSON(1105)
	errorUnknownPOI             = errorJSON(1106)
	errorNoPOIInProfile         = errorJSON(1107)

	errorRequestNotExist     = errorJSON(1200)
	errorMultipleRequestMade = errorJSON(1201)

	errorPOIListNotFound      = errorJSON(1300)
	errorPOIListMissmatch     = errorJSON(1301)
	errorEmptyPOIResourceName = errorJSON(1303)
)

type ErrorResponse struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

// errorJSON converts an error code to a standardized error object
func errorJSON(code int64) ErrorResponse {
	var message string
	if msg, ok := errorMessageMap[code]; ok {
		message = msg
	} else {
		message = "unknown"
	}

	return ErrorResponse{
		Code:    code,
		Message: message,
	}
}
