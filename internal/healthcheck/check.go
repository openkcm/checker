package healthcheck

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"

	"github.com/openkcm/checker/internal/config"
)

func (ch *CachedResponses) process(
	ctx context.Context,
	cfg *config.Healthcheck,
	resultCollector *ResultCollector,
) {
	wg := sync.WaitGroup{}

	service := &cfg.Cluster
	if service.Enabled && len(service.Resources) > 0 {
		wg.Go(func() {
			ch.processResources(ctx, service, "services", verifyServiceResource, resultCollector)
		})
	}

	k8s := &cfg.Kubernetes
	if k8s.Enabled && len(k8s.Resources) > 0 {
		wg.Go(func() {
			ch.processResources(ctx, k8s, "kubernetes", verifyK8SResource, resultCollector)
		})
	}

	linkerd := &cfg.Linkerd
	if linkerd.Enabled {
		wg.Go(func() {
			ch.processLinkerdResources(ctx, linkerd, resultCollector)
		})
	}

	wg.Wait()
}

func (ch *CachedResponses) processResources(
	ctx context.Context,
	cfg *config.Domain,
	tag string,
	verifyResource func(context.Context, *config.Resource) (*Response, int),
	resultCollector *ResultCollector,
) {
	if len(cfg.Resources) == 0 {
		return
	}

	resultCollector.Data.Store(tag, make([]*Response, 0))

	wg := sync.WaitGroup{}
	wg.Add(len(cfg.Resources))

	for _, h := range cfg.Resources {
		go func(rc *config.Resource, resultCollector *ResultCollector) {
			defer wg.Done()

			resp, respStatus := verifyResource(ctx, rc)

			value, _ := resultCollector.Data.Load(tag)
			l, _ := value.([]*Response)
			l = append(l, resp)
			resultCollector.Data.Store(tag, l)

			retryStatus := ch.updateRetryState(
				h.Retry.MaxRetries,
				hashMultipleStrings(tag, h.Name, h.URL),
				respStatus,
			)
			if retryStatus == http.StatusOK && respStatus != http.StatusOK {
				resp.Status = OK_TOLERATED_FAILURE_ON_RETRY
			}

			if retryStatus != http.StatusOK {
				resultCollector.Status.Store(retryStatus)
			}
		}(&h, resultCollector)
	}

	wg.Wait()
}

func (ch *CachedResponses) processLinkerdResources(
	ctx context.Context,
	linkerd *config.Linkerd,
	resultCollector *ResultCollector,
) {
	resp, respStatus := verifyLinkerd(ctx, linkerd)
	resultCollector.Data.Store(linkerd.Tag, resp)

	retryStatus := ch.updateRetryState(
		linkerd.Retry.MaxRetries,
		hashMultipleStrings(linkerd.Tag, linkerd.ControlPlaneNamespace, linkerd.DataPlaneNamespace, linkerd.CNINamespace),
		respStatus,
	)
	if retryStatus == http.StatusOK && respStatus != http.StatusOK {
		resp.Status = OK_TOLERATED_FAILURE_ON_RETRY
	}

	if retryStatus != http.StatusOK {
		resultCollector.Status.Store(retryStatus)
	}
}

// updateRetryState updates the retry map and returns the status pointer if the response failed and max retries are exceeded.
func (ch *CachedResponses) updateRetryState(maxRetries int, key string, respStatus int) int {
	// Fast path: if retries are disabled entirely
	if maxRetries <= 0 {
		return respStatus
	}

	// If the request was successful, reset the state and return OK
	if respStatus == http.StatusOK {
		ch.SetRetry(key, 0)
		return http.StatusOK
	}

	// At this point, the response is NOT OK.
	// Retrieve the current failure count (defaults to 0 if not found).

	v := ch.GetRetry(key)

	// If we have hit or exceeded the max retries
	if v >= maxRetries {
		// Return the actual error status because the limit is reached
		return respStatus
	}

	// Limit is not reached yet.
	// Increment the failure count and save it back to the map.
	v++
	ch.SetRetry(key, v)

	// Mask the error and pretend everything is "OK" for now.
	return http.StatusOK
}

func hashMultipleStrings(stringsToHash ...string) string {
	hash := sha256.New()

	// Buffer to hold the 64-bit length of each string
	delimiter := []byte{0}
	lengthBytes := make([]byte, 8)

	for _, str := range stringsToHash {
		if strings.TrimSpace(str) == "" {
			continue
		}

		// 1. Convert the string length to bytes and write it
		binary.LittleEndian.PutUint64(lengthBytes, uint64(len(str)))
		hash.Write(lengthBytes)

		// 2. Write the actual string content
		hash.Write([]byte(str))
		hash.Write(delimiter)
	}

	hashBytes := hash.Sum(nil)

	return hex.EncodeToString(hashBytes)
}
