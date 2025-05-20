package healthcheck

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"

	"github.com/openkcm/checker/internal/config"
)

func verifyKubernetesResource(ctx context.Context, rc *config.KubernetesResource) (*Response, int) {
	status := http.StatusOK
	errors := make([]ErrorResponse, 0)

	response := &Response{
		Name:   rc.Name,
		URL:    rc.URL,
		Status: OK,
	}

	defer func() {
		if len(errors) > 0 {
			status = http.StatusServiceUnavailable
			response.Errors = errors
			response.Status = NOTOK
		}
	}()

	var k8sConfig *rest.Config
	var err error

	// Check for KUBECONFIG or fallback to in-cluster
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		k8sConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Bad Request",
		})
		return response, status
	}

	transportConfig, err := k8sConfig.TransportConfig()
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Bad Request",
		})
		return response, status
	}

	rt, err := transport.New(transportConfig)
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Bad Request",
		})
		return response, status
	}

	client := &http.Client{Transport: rt, Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k8sConfig.Host+rc.URL, nil)
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Bad Request",
		})
		return response, status
	}

	resp, err := client.Do(req)
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Reading Response Body",
		})
		return response, status
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
		return response, status
	}

	errors = verifyChecks(rc.Checks, body, []byte(resp.Status), errors)

	return response, status
}
