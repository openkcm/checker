package healthcheck

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
