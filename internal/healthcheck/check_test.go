package healthcheck

import (
	"context"
	"net/http"
	"testing"

	"github.com/openkcm/checker/internal/config"
)

func TestHashMultipleStrings(t *testing.T) {
	h1 := hashMultipleStrings("a", "b", "c")

	h2 := hashMultipleStrings("a", "b", "c")
	if h1 != h2 {
		t.Error("hash should be deterministic")
	}

	// Blank entries are skipped, so these must differ from the full hash.
	if hashMultipleStrings("a", "", "c") == h1 {
		t.Error("blank fields should change the hash")
	}

	// Avoid length-extension style collisions between different groupings.
	if hashMultipleStrings("ab", "c") == hashMultipleStrings("a", "bc") {
		t.Error("length-prefixed hashing should avoid collisions")
	}
}

func TestUpdateRetryStateRetriesDisabled(t *testing.T) {
	ch := &CachedResponses{}

	if got := ch.updateRetryState(0, "k", http.StatusServiceUnavailable); got != http.StatusServiceUnavailable {
		t.Errorf("got %d, want 503 when retries disabled", got)
	}
}

func TestUpdateRetryStateSuccessResets(t *testing.T) {
	ch := &CachedResponses{}
	ch.SetRetry("k", 3)

	if got := ch.updateRetryState(5, "k", http.StatusOK); got != http.StatusOK {
		t.Errorf("got %d, want 200", got)
	}

	if v := ch.GetRetry("k"); v != 0 {
		t.Errorf("retry counter = %d, want reset to 0", v)
	}
}

func TestUpdateRetryStateToleratesUntilMax(t *testing.T) {
	ch := &CachedResponses{}
	key := "k"

	// First failure with maxRetries=2: tolerated (masked as OK), counter -> 1.
	if got := ch.updateRetryState(2, key, http.StatusServiceUnavailable); got != http.StatusOK {
		t.Errorf("first failure: got %d, want 200 (tolerated)", got)
	}

	if v := ch.GetRetry(key); v != 1 {
		t.Errorf("counter = %d, want 1", v)
	}

	// Second failure: tolerated, counter -> 2.
	if got := ch.updateRetryState(2, key, http.StatusServiceUnavailable); got != http.StatusOK {
		t.Errorf("second failure: got %d, want 200 (tolerated)", got)
	}

	// Third failure: limit reached, real error surfaced.
	if got := ch.updateRetryState(2, key, http.StatusServiceUnavailable); got != http.StatusServiceUnavailable {
		t.Errorf("third failure: got %d, want 503", got)
	}
}

func TestProcessResourcesEmpty(t *testing.T) {
	ch := &CachedResponses{}
	rc := newResultCollector()

	// No resources -> early return, nothing stored.
	ch.processResources(context.Background(), &config.Domain{Tag: "cluster"}, verifyServiceResource, rc)

	if _, ok := rc.Data.Load("cluster"); ok {
		t.Error("expected no data stored for empty resources")
	}
}

func TestProcessResourcesSuccessAndFailure(t *testing.T) {
	ch := &CachedResponses{}
	rc := newResultCollector()

	cfg := &config.Domain{
		Tag: "cluster",
		Resources: []config.Resource{
			{Name: "ok", URL: "http://ok"},
			{Name: "bad", URL: "http://bad"},
		},
	}

	// Stub verifier: "bad" resource fails.
	verify := func(_ context.Context, rc *config.Resource) (*Response, int) {
		if rc.Name == "bad" {
			return &Response{Name: rc.Name, Status: NOTOK}, http.StatusServiceUnavailable
		}

		return &Response{Name: rc.Name, Status: OK}, http.StatusOK
	}

	ch.processResources(context.Background(), cfg, verify, rc)

	value, ok := rc.Data.Load("cluster")
	if !ok {
		t.Fatal("expected data for cluster tag")
	}

	responses, ok := value.([]*Response)
	if !ok {
		t.Fatalf("stored value has type %T, want []*Response", value)
	}

	if got := len(responses); got != 2 {
		t.Errorf("stored %d responses, want 2", got)
	}

	if rc.Status.Read() != http.StatusServiceUnavailable {
		t.Errorf("collector status = %d, want 503", rc.Status.Read())
	}
}

func TestProcessResourcesToleratedOnRetry(t *testing.T) {
	ch := &CachedResponses{}
	rc := newResultCollector()

	cfg := &config.Domain{
		Tag:       "cluster",
		Resources: []config.Resource{{Name: "flaky", URL: "http://flaky", Retry: config.Retry{MaxRetries: 3}}},
	}

	verify := func(context.Context, *config.Resource) (*Response, int) {
		return &Response{Name: "flaky", Status: NOTOK}, http.StatusServiceUnavailable
	}

	ch.processResources(context.Background(), cfg, verify, rc)

	value, _ := rc.Data.Load("cluster")

	responses, ok := value.([]*Response)
	if !ok {
		t.Fatalf("stored value has type %T, want []*Response", value)
	}

	resp := responses[0]
	if resp.Status != OK_TOLERATED_FAILURE_ON_RETRY {
		t.Errorf("status = %q, want tolerated-failure marker", resp.Status)
	}

	if rc.Status.Read() != http.StatusOK {
		t.Errorf("collector status = %d, want 200 (tolerated)", rc.Status.Read())
	}
}

func TestProcessDispatchesEnabledDomains(t *testing.T) {
	ch := &CachedResponses{}
	rc := newResultCollector()

	cfg := &config.Healthcheck{
		Cluster: config.Domain{
			Enabled:   true,
			Resources: []config.Resource{{Name: "c", URL: "://bad-url"}},
		},
		Kubernetes: config.Domain{
			Enabled:   true,
			Resources: []config.Resource{{Name: "k", URL: "/healthz"}},
		},
	}

	ch.process(context.Background(), cfg, rc)

	// Default tags are applied when empty.
	if cfg.Cluster.Tag != "cluster" {
		t.Errorf("cluster tag = %q, want cluster", cfg.Cluster.Tag)
	}

	if cfg.Kubernetes.Tag != "kubernetes" {
		t.Errorf("kubernetes tag = %q, want kubernetes", cfg.Kubernetes.Tag)
	}

	if _, ok := rc.Data.Load("cluster"); !ok {
		t.Error("expected cluster results to be collected")
	}
}

func TestProcessSkipsDisabledDomains(t *testing.T) {
	ch := &CachedResponses{}
	rc := newResultCollector()

	// All domains disabled -> nothing collected, status stays OK.
	ch.process(context.Background(), &config.Healthcheck{}, rc)

	if rc.Status.Read() != http.StatusOK {
		t.Errorf("status = %d, want 200", rc.Status.Read())
	}
}

func TestProcessLinkerdResources(t *testing.T) {
	t.Setenv("KUBECONFIG", "/nonexistent/kubeconfig.yaml")

	ch := &CachedResponses{}
	rc := newResultCollector()

	linkerd := &config.Linkerd{
		Tag:                   "linkerd",
		ControlPlaneNamespace: "linkerd",
		Checks:                []string{},
	}

	// Without a kube API client available, the check fails and status is set.
	ch.processLinkerdResources(context.Background(), linkerd, rc)

	if _, ok := rc.Data.Load("linkerd"); !ok {
		t.Error("expected linkerd result to be stored")
	}

	if rc.Status.Read() != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rc.Status.Read())
	}
}
