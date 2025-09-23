package healthcheck

import (
	"context"
	"sync"
	"time"

	"github.com/openkcm/checker/internal/config"
)

type CachedResponses struct {
	mu sync.RWMutex

	status   int
	response map[string]any
}

func NewCachedResponses(ctx context.Context, cfg *config.Healthcheck) *CachedResponses {
	cache := &CachedResponses{}
	go func(cfg *config.Healthcheck, ch *CachedResponses) {
		ticker := time.NewTicker(cfg.RefreshDuration)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ch.refresh(ctx, cfg)
			}
		}
	}(cfg, cache)

	return cache
}

func (ch *CachedResponses) Status() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return ch.status
}

func (ch *CachedResponses) Response() map[string]any {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return ch.response
}

func (ch *CachedResponses) refresh(ctx context.Context, cfg *config.Healthcheck) {
	response, status := Do(ctx, cfg)

	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.status = status
	ch.response = response
}
