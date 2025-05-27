package version

const OK = "OK"
const NOTOK = "NOT OK"

type Response struct {
	URL    string         `json:"url,omitempty"`
	Error  *ErrorResponse `json:"error,omitempty"`
	Status string         `json:"status,omitempty"`
	Result any            `json:"body,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}
