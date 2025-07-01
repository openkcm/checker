package healthcheck

import (
	"context"
	"sync"
	"time"

	"github.com/openkcm/checker/internal/config"
)

type CachedResponses struct {
	mu sync.Mutex

	status   int
	response map[string]any
}

func NewCachedResponses(ctx context.Context, cfg *config.Healthcheck) *CachedResponses {
	cache := &CachedResponses{}
	go func(cfg *config.Healthcheck, ch *CachedResponses) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				ch.refresh(ctx, cfg)
				time.Sleep(cfg.RefreshDuration)
			}
		}

	}(cfg, cache)
	return cache
}

func (ch *CachedResponses) refresh(ctx context.Context, cfg *config.Healthcheck) {
	response, status := Do(ctx, cfg)
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.status = status
	ch.response = response
}

func (ch *CachedResponses) Status() int {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.status
}

func (ch *CachedResponses) Response() map[string]any {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.response
}
