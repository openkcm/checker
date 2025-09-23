package version

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/openkcm/checker/internal/config"
)

func Do(ctx context.Context, cfg *config.Versions) (map[string]any, int) {
	status := http.StatusOK
	response := map[string]any{}

	for _, svc := range cfg.Resources {
		res := &Response{
			URL:    svc.URL,
			Status: OK,
		}

		body, err := call(ctx, svc)
		if err != nil {
			res.Status = NOTOK
			res.Error = &ErrorResponse{
				Error:   err.Error(),
				Message: "Failed to call: " + svc.URL,
			}
		} else {
			res.Result = json.RawMessage(body)
		}

		response[svc.Name] = res
	}

	return response, status
}

func call(ctx context.Context, svc *config.ServiceResource) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(b io.ReadCloser) {
		_ = b.Close()
	}(resp.Body)

	return io.ReadAll(resp.Body)
}
