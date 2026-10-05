package healthcheck

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/openkcm/checker/internal/config"
	"github.com/openkcm/checker/internal/utils"
)

// newResultCollector builds a ResultCollector initialised the same way refresh() does.
func newResultCollector() *ResultCollector {
	return &ResultCollector{
		Status: *utils.NewContainerWithDefault[int](http.StatusOK),
		Data:   sync.Map{},
	}
}

func TestGetSetRetry(t *testing.T) {
	ch := &CachedResponses{}

	if v := ch.GetRetry("missing"); v != 0 {
		t.Errorf("missing key = %d, want 0", v)
	}

	ch.SetRetry("k", 7)

	if v := ch.GetRetry("k"); v != 7 {
		t.Errorf("k = %d, want 7", v)
	}
}

func TestGetRetryWrongType(t *testing.T) {
	ch := &CachedResponses{}
	ch.retry.Store("k", "not-an-int")

	if v := ch.GetRetry("k"); v != 0 {
		t.Errorf("non-int value should yield 0, got %d", v)
	}
}

func TestStatusAndResponseAccessors(t *testing.T) {
	ch := &CachedResponses{}
	ch.mu.Lock()
	ch.status = http.StatusServiceUnavailable
	ch.response = map[string]any{"a": 1}
	ch.mu.Unlock()

	if ch.Status() != http.StatusServiceUnavailable {
		t.Errorf("Status() = %d, want 503", ch.Status())
	}

	if ch.Response()["a"] != 1 {
		t.Errorf("Response() = %v, want a=1", ch.Response())
	}
}

func TestRefreshPopulatesResponse(t *testing.T) {
	ch := &CachedResponses{}

	cfg := &config.Healthcheck{
		Cluster: config.Domain{
			Enabled: true,
			Tag:     "cluster",
			Resources: []config.Resource{
				{Name: "bad", URL: "://bad-url"},
			},
		},
	}

	ch.refresh(context.Background(), cfg)

	if _, ok := ch.Response()["cluster"]; !ok {
		t.Error("expected cluster key in refreshed response")
	}

	if ch.Status() != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", ch.Status())
	}
}

func TestNewCachedResponsesRefreshesAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := &config.Healthcheck{
		RefreshDuration: 10 * time.Millisecond,
		Cluster: config.Domain{
			Enabled:   true,
			Tag:       "cluster",
			Resources: []config.Resource{{Name: "bad", URL: "://bad-url"}},
		},
	}

	cache := NewCachedResponses(ctx, cfg)

	// Wait for the initial synchronous-ish refresh to populate the response.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, ok := cache.Response()["cluster"]; ok {
			break
		}

		time.Sleep(5 * time.Millisecond)
	}

	if _, ok := cache.Response()["cluster"]; !ok {
		t.Error("expected background refresh to populate response")
	}

	cancel()
	// Give the goroutine a moment to observe cancellation and return.
	time.Sleep(20 * time.Millisecond)
}
