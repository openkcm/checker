package healthcheck

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/openkcm/checker/internal/config"
	"github.com/openkcm/checker/internal/utils"
)

type CachedResponses struct {
	mu       sync.RWMutex
	status   int
	response map[string]any

	retry sync.Map
}

type ResultCollector struct {
	Status utils.Container[int]
	Data   sync.Map
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
	collector := &ResultCollector{
		Status: *utils.NewContainerWithDefault[int](http.StatusOK),
		Data:   sync.Map{},
	}

	ch.process(ctx, cfg, collector)

	response := map[string]any{}

	collector.Data.Range(func(key, value any) bool {
		val, ok := key.(string)
		if !ok {
			return true
		}

		response[val] = value

		return true
	})

	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.response = response
	ch.status = collector.Status.Read()
}
