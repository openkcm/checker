package version

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/openkcm/checker/internal/config"
)

func Do(ctx context.Context, cfg *config.Versions) (map[string]any, int) {
	status := http.StatusOK

	mu := &sync.Mutex{}
	response := map[string]any{}

	wg := sync.WaitGroup{}
	wg.Add(len(cfg.Resources))

	client := &http.Client{Timeout: 5 * time.Second}

	for _, svc := range cfg.Resources {
		go func(mu *sync.Mutex, response map[string]any) {
			defer wg.Done()

			res := &Response{
				URL:    svc.URL,
				Status: OK,
			}

			body, err := call(ctx, client, svc)
			if err != nil {
				res.Status = NOTOK
				res.Error = &ErrorResponse{
					Error:   err.Error(),
					Message: "Failed to call: " + svc.URL,
				}
			} else {
				res.Result = json.RawMessage(body)
			}

			mu.Lock()
			defer mu.Unlock()
			response[svc.Name] = res
		}(mu, response)

	}

	wg.Wait()

	return response, status
}

func call(ctx context.Context, client *http.Client, svc *config.ServiceResource) ([]byte, error) {
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
