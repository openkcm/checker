package versions

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/openkcm/common-sdk/pkg/utils"

	"github.com/openkcm/checker/internal/config"
)

func Query(ctx context.Context, cfg *config.Versions) map[string]any {
	lenResources := len(cfg.Resources)

	response := map[string]any{}

	if lenResources == 0 {
		return response
	}

	mu := &sync.Mutex{}

	wg := sync.WaitGroup{}
	wg.Add(lenResources)

	client := &http.Client{Timeout: cfg.Timeout}

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
				unmarshalValue(string(body), res)
			}

			mu.Lock()
			defer mu.Unlock()

			response[svc.Name] = res
		}(mu, response)
	}

	wg.Wait()

	return response
}

func unmarshalValue(value string, res *Response) {
	jsonVersion, err := utils.ExtractFromComplexValue(value)
	if err != nil {
		res.Status = NOTOK
		res.Error = &ErrorResponse{
			Error:   err.Error(),
			Message: "Failed to decode the response: " + value,
		}
		res.Result = nil
	} else {
		res.Result = map[string]any{}

		err = json.Unmarshal([]byte(jsonVersion), &res.Result)
		if err != nil {
			res.Status = NOTOK
			res.Error = &ErrorResponse{
				Error:   err.Error(),
				Message: "Failed to unmarshal the following response: " + jsonVersion,
			}
			res.Result = nil
		}
	}
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
