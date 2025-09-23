package healthcheck

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/openkcm/checker/internal/config"
)

func verifyClusterResource(ctx context.Context, rc *config.ClusterResource) (*Response, int) {
	errors := make([]ErrorResponse, 0)
	response := &Response{
		Name:   rc.Name,
		URL:    rc.URL,
		Status: OK,
	}

	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rc.URL, nil)
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Bad Request",
		})
		response.Errors = errors
		response.Status = NOTOK

		return response, http.StatusServiceUnavailable
	}

	resp, err := client.Do(req)
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Reading Response Body",
		})
		response.Errors = errors
		response.Status = NOTOK

		return response, http.StatusServiceUnavailable
	}
	defer func(b io.ReadCloser) {
		_ = b.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Reading Response Body",
		})
		response.Errors = errors
		response.Status = NOTOK

		return response, http.StatusServiceUnavailable
	}

	status := http.StatusOK

	errors = verifyChecks(rc.Checks, body, []byte(resp.Status), errors)
	if len(errors) > 0 {
		status = http.StatusServiceUnavailable
		response.Errors = errors
		response.Status = NOTOK
	}

	return response, status
}
