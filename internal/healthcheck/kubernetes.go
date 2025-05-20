package healthcheck

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/openkcm/checker/internal/config"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"
)

func verifyKubernetesResource(rc *config.KubernetesResource) (ret Response, status int) {
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
		return
	}

	transportConfig, err := k8sConfig.TransportConfig()
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Bad Request",
		})
		return
	}

	rt, err := transport.New(transportConfig)
	if err != nil {
		errors = append(errors, ErrorResponse{
			Message: err.Error(),
			Error:   "Bad Request",
		})
		return
	}

	client := &http.Client{Transport: rt, Timeout: 5 * time.Second}

	resp, err := client.Get(k8sConfig.Host + rc.URL)
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
