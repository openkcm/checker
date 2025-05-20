package healthcheck

import (
	"io"
	"net/http"
	"time"

	"github.com/openkcm/checker/internal/config"
)

func verifyClusterResource(rc *config.ClusterResource) (ret Response, status int) {
	status = http.StatusOK
	errors := make([]ErrorResponse, 0)
	ret = Response{
		Name:   rc.Name,
		URL:    rc.URL,
		Status: "OK",
	}

	defer func() {
		if len(errors) > 0 {
			status = http.StatusServiceUnavailable
			ret.Errors = errors
			ret.Status = "NOT OK"
		}
	}()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(rc.URL)
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Bad Request",
		})

		return
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Reading Response Body",
		})
		return
	}

	verifyChecks(rc.Checks, body, []byte(resp.Status), errors)

	return
}
