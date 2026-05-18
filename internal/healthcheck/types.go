package healthcheck

const (
	OK                            = "OK"
	OK_TOLERATED_FAILURE_ON_RETRY = "OK [FAILURE TOLERATED ON RETRY]"
	NOTOK                         = "NOT OK"
	ERR_BAD_REQUEST               = "Bad Request"
	ERR_READING_RESPONSE_BODY     = "Reading Response Body"
)

type Response struct {
	Name   string          `json:"name,omitempty"`
	URL    string          `json:"url,omitempty"`
	Errors []ErrorResponse `json:"errors,omitempty"`
	Status string          `json:"status,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}
