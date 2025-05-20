package healthcheck

import (
	"context"
	"net/http"
	"sync"

	"github.com/openkcm/checker/internal/config"
)

func Do(ctx context.Context, cfg *config.Healthcheck) (map[string]any, int) {
	status := http.StatusOK
	response := map[string]any{}

	wg := sync.WaitGroup{}

	cluster := cfg.Cluster
	if cluster.Enabled {
		clusterMu := &sync.Mutex{}
		response[cluster.Tag] = make([]*Response, 0)

		wg.Add(len(cluster.Resources))
		for _, h := range cluster.Resources {
			go func(rc *config.ClusterResource, mu *sync.Mutex, m map[string]any) {
				defer wg.Done()

				resp, respStatus := verifyClusterResource(ctx, rc)
				mu.Lock()
				defer mu.Unlock()

				l, _ := m[cluster.Tag].([]*Response)
				m[cluster.Tag] = append(l, resp)

				if respStatus != http.StatusOK {
					status = respStatus
				}
			}(&h, clusterMu, response)
		}
	}

	k8s := cfg.Kubernetes
	if k8s.Enabled {
		k8Mu := &sync.Mutex{}
		response[k8s.Tag] = make([]*Response, 0)

		wg.Add(len(k8s.Resources))
		for _, h := range k8s.Resources {
			go func(rc *config.KubernetesResource, mu *sync.Mutex, m map[string]any) {
				defer wg.Done()

				resp, respStatus := verifyKubernetesResource(ctx, rc)
				mu.Lock()
				defer mu.Unlock()
				l, _ := m[k8s.Tag].([]*Response)
				m[k8s.Tag] = append(l, resp)

				if respStatus != http.StatusOK {
					status = respStatus
				}
			}(&h, k8Mu, response)
		}
	}

	linkerd := cfg.Linkerd
	if linkerd.Enabled {
		resp, respStatus := verifyLinkerd(ctx, &linkerd)
		response[linkerd.Tag] = resp

		if respStatus != http.StatusOK {
			status = respStatus
		}
	}

	wg.Wait()

	return response, status
}
