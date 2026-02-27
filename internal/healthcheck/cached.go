package healthcheck

import (
	"context"
	"sync"
	"time"

	"github.com/openkcm/checker/internal/config"
)

type CachedResponses struct {
	mu       sync.RWMutex
	status   int
	response map[string]any

	retry sync.Map
}

func NewCachedResponses(ctx context.Context, cfg *config.Healthcheck) *CachedResponses {
	cache := &CachedResponses{
		retry: sync.Map{},
	}
	go func(cfg *config.Healthcheck, ch *CachedResponses) {
		ticker := time.NewTicker(cfg.RefreshDuration)
		defer ticker.Stop()

		ch.refresh(ctx, cfg)

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

func (ch *CachedResponses) SetRetry(key string, value int) {
	ch.retry.Store(key, value)
}

func (ch *CachedResponses) GetRetry(key string) int {
	val, ok := ch.retry.Load(key)
	if !ok {
		return 0
	}

	if val, ok := val.(int); ok {
		return val
	}

	return 0
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
	response, status := ch.Do(ctx, cfg)

	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.status = status
	ch.response = response
}
