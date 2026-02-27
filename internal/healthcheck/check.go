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
	"github.com/openkcm/checker/internal/utils"
)

func (ch *CachedResponses) Do(ctx context.Context, cfg *config.Healthcheck) (map[string]any, int) {
	statusContainer := utils.NewContainerWithDefault[int](http.StatusOK)

	response := map[string]any{}

	wg := sync.WaitGroup{}

	cluster := cfg.Cluster
	if cluster.Enabled {
		clusterMu := &sync.Mutex{}
		response[cluster.Tag] = make([]*Response, 0)

		wg.Add(len(cluster.Resources))

		for _, h := range cluster.Resources {
			go func(rc *config.ClusterResource, mu *sync.Mutex, m map[string]any, status *utils.Container[int]) {
				defer wg.Done()

				resp, respStatus := verifyClusterResource(ctx, rc)

				mu.Lock()
				defer mu.Unlock()

				l, _ := m[cluster.Tag].([]*Response)
				m[cluster.Tag] = append(l, resp)

				retryStatus := ch.updateRetryState(
					h.Retry.MaxRetries,
					hashMultipleStrings(cluster.Tag, h.Name, h.URL),
					respStatus,
				)
				if retryStatus == http.StatusOK && respStatus != http.StatusOK {
					resp.Status = OK_TOLERATED_FAILURE_ON_RETRY
				}

				if retryStatus != http.StatusOK {
					status.Store(retryStatus)
				}
			}(&h, clusterMu, response, statusContainer)
		}
	}

	k8s := cfg.Kubernetes
	if k8s.Enabled {
		k8Mu := &sync.Mutex{}
		response[k8s.Tag] = make([]*Response, 0)

		wg.Add(len(k8s.Resources))

		for _, h := range k8s.Resources {
			go func(rc *config.KubernetesResource, mu *sync.Mutex, m map[string]any, status *utils.Container[int]) {
				defer wg.Done()

				resp, respStatus := verifyKubernetesResource(ctx, rc)

				mu.Lock()
				defer mu.Unlock()

				l, _ := m[k8s.Tag].([]*Response)
				m[k8s.Tag] = append(l, resp)

				retryStatus := ch.updateRetryState(
					h.Retry.MaxRetries,
					hashMultipleStrings(k8s.Tag, h.Name, h.URL),
					respStatus,
				)
				if retryStatus == http.StatusOK && respStatus != http.StatusOK {
					resp.Status = OK_TOLERATED_FAILURE_ON_RETRY
				}

				if retryStatus != http.StatusOK {
					status.Store(retryStatus)
				}
			}(&h, k8Mu, response, statusContainer)
		}
	}

	linkerd := cfg.Linkerd
	if linkerd.Enabled {
		resp, respStatus := verifyLinkerd(ctx, &linkerd)
		response[linkerd.Tag] = resp

		retryStatus := ch.updateRetryState(
			linkerd.Retry.MaxRetries,
			hashMultipleStrings(linkerd.Tag, linkerd.ControlPlaneNamespace, linkerd.DataPlaneNamespace, linkerd.CNINamespace),
			respStatus,
		)
		if retryStatus == http.StatusOK && respStatus != http.StatusOK {
			resp.Status = OK_TOLERATED_FAILURE_ON_RETRY
		}

		if retryStatus != http.StatusOK {
			statusContainer.Store(retryStatus)
		}
	}

	wg.Wait()

	return response, statusContainer.Read()
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
