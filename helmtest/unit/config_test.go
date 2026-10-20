//go:build helmtest

package unit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/linkerd/linkerd2/pkg/healthcheck"
	"github.com/openkcm/common-sdk/pkg/commoncfg"
	"go.yaml.in/yaml/v3"

	"github.com/openkcm/checker/internal/config"
	checks "github.com/openkcm/checker/internal/healthcheck"
)

// loadConfig writes the rendered config.yaml to a directory and loads it the way the
// Checker does at startup (cmd/checker/main.go): commoncfg.LoadConfig, which rejects
// unknown keys and then applies the `default` tags of the configuration types.
func loadConfig(t *testing.T, configYAML string) (*config.Config, error) {
	t.Helper()

	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(configYAML), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}

	err = commoncfg.LoadConfig(cfg, map[string]any{}, dir)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadValues renders the chart with the arguments and loads the configuration the way the
// Checker does at startup.
func loadValues(t *testing.T, args ...string) *config.Config {
	t.Helper()

	cfg, err := loadConfig(t, renderConfigYAML(t, args...))
	if err != nil {
		t.Fatalf("the Checker does not load the configuration: %v", err)
	}

	return cfg
}

// readmeValues writes the example values file of the chart's README to a file and
// returns its path.
func readmeValues(t *testing.T) string {
	t.Helper()

	readme, err := os.ReadFile(chartPath + "/README.md")
	if err != nil {
		t.Fatal(err)
	}

	_, rest, found := strings.Cut(string(readme), "Create a values file")
	if found {
		_, rest, found = strings.Cut(rest, "```yaml\n")
	}

	example, _, closed := strings.Cut(rest, "```")
	if !found || !closed {
		t.Fatal("README.md has no complete values file example")
	}

	return writeValues(t, example)
}

// TestConfigStrict checks that the Checker loads the configuration rendered from the default
// values, the README example and every example documented under `config` in values.yaml
// (testdata/values-all-config*.yaml).
func TestConfigStrict(t *testing.T) {
	tests := []struct {
		name string
		args func(t *testing.T) []string
	}{
		{
			name: "default values",
			args: func(*testing.T) []string { return nil },
		},
		{
			name: "values file example of the README",
			args: func(t *testing.T) []string {
				t.Helper()
				return []string{"-f", readmeValues(t)}
			},
		},
		{
			name: "every documented setting, first example of each alternative",
			args: func(*testing.T) []string { return []string{"-f", "testdata/values-all-config.yaml"} },
		},
		{
			name: "second example of each alternative",
			args: func(*testing.T) []string { return []string{"-f", "testdata/values-all-config-alt1.yaml"} },
		},
		{
			name: "third example of each alternative",
			args: func(*testing.T) []string { return []string{"-f", "testdata/values-all-config-alt2.yaml"} },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadValues(t, tt.args(t)...)
		})
	}

	t.Run("the loader rejects unknown keys", func(t *testing.T) {
		_, err := loadConfig(t, renderConfigYAML(t, "--set", "config.healthcheck.notACheckerSetting=1"))
		if err == nil || !strings.Contains(err.Error(), "invalid keys") {
			t.Fatalf("want an error with invalid keys, got %v", err)
		}
	})
}

// TestConfigDefaults checks the configuration that the default values give the Checker, as
// values.yaml describes it.
func TestConfigDefaults(t *testing.T) {
	cfg := loadValues(t)

	if cfg.Application.Name != "checker" || cfg.Application.Environment != "production" {
		t.Errorf("application: got %+v", cfg.Application)
	}

	if cfg.Server.Address != ":8080" || cfg.Server.ShutdownTimeout != 5*time.Second {
		t.Errorf("server: got %+v", cfg.Server)
	}

	if !cfg.Status.Enabled || cfg.Status.Address != ":8888" || cfg.Status.Profiling {
		t.Errorf("status: got %+v", cfg.Status)
	}

	hc := cfg.Healthcheck
	if !hc.Enabled || hc.Endpoint != "/healthz" || hc.Name != "healthcheck" ||
		hc.RefreshDuration != 5*time.Second || hc.MaskURLs {
		t.Errorf("healthcheck: got enabled %v, endpoint %q, name %q, refreshDuration %v, maskURLs %v",
			hc.Enabled, hc.Endpoint, hc.Name, hc.RefreshDuration, hc.MaskURLs)
	}

	if hc.Cluster.Enabled || !hc.Kubernetes.Enabled || !hc.Linkerd.Enabled {
		t.Errorf("enabled groups: cluster %v, kubernetes %v, linkerd %v",
			hc.Cluster.Enabled, hc.Kubernetes.Enabled, hc.Linkerd.Enabled)
	}

	for _, resource := range slices.Concat(hc.Cluster.Resources, hc.Kubernetes.Resources) {
		if len(resource.Checks) == 0 {
			t.Errorf("resource %s has no checks", resource.Name)
		}
	}

	if len(cfg.Healthchecks) != 0 {
		t.Errorf("healthchecks: got %d entries", len(cfg.Healthchecks))
	}

	if cfg.Versions.Enabled || cfg.Versions.Timeout != 5*time.Second {
		t.Errorf("versions: got %+v", cfg.Versions)
	}

	for name, signal := range map[string]bool{
		"logs":    cfg.Telemetry.Logs.Enabled,
		"traces":  cfg.Telemetry.Traces.Enabled,
		"metrics": cfg.Telemetry.Metrics.Enabled,
	} {
		if signal {
			t.Errorf("telemetry %s is enabled", name)
		}
	}
}

// TestConfigPassThrough checks that everything under config reaches config.yaml (helper
// checker.config), except the chart-only key isImmutable, and how the application block is
// built.
func TestConfigPassThrough(t *testing.T) {
	t.Run("sections of the default values", func(t *testing.T) {
		var root map[string]any

		err := yaml.Unmarshal([]byte(renderConfigYAML(t)), &root)
		if err != nil {
			t.Fatal(err)
		}

		want := []string{
			"application", "healthcheck", "healthchecks", "logger", "server", "status",
			"telemetry", "versions",
		}

		got := make([]string, 0, len(root))
		for key := range root {
			got = append(got, key)
		}

		slices.Sort(got)

		if !slices.Equal(got, want) {
			t.Errorf("sections of config.yaml: got %v, want %v", got, want)
		}
	})

	t.Run("isImmutable is not written", func(t *testing.T) {
		cfg := renderConfigYAML(t, "--set", "config.isImmutable=true")
		if strings.Contains(cfg, "isImmutable") {
			t.Errorf("config.yaml contains isImmutable:\n%s", cfg)
		}
	})

	t.Run("application from environment, labels and nameOverride", func(t *testing.T) {
		cfg := loadValues(t, "--set", "nameOverride=health", "--set", "config.environment=staging",
			"--set", "config.labels.team=key-management")

		app := cfg.Application
		if app.Name != "health" || app.Environment != "staging" ||
			!reflect.DeepEqual(app.Labels, map[string]string{"team": "key-management"}) {
			t.Errorf("application: got name %q, environment %q, labels %v",
				app.Name, app.Environment, app.Labels)
		}
	})

	t.Run("application block takes precedence", func(t *testing.T) {
		cfg := loadValues(t, "--set", "config.environment=staging",
			"--set", "config.application.name=checker-emea", "--set", "config.application.environment=dev")

		if cfg.Application.Name != "checker-emea" || cfg.Application.Environment != "dev" {
			t.Errorf("application: got %+v", cfg.Application)
		}
	})
}

// documentedCategories returns the categories of `linkerd check` that values.yaml lists at
// config.healthcheck.linkerd.checks.
func documentedCategories(t *testing.T) []string {
	t.Helper()

	values, err := os.ReadFile(chartPath + "/values.yaml")
	if err != nil {
		t.Fatal(err)
	}

	_, rest, found := strings.Cut(string(values), "Categories and what they need:\n")
	list, _, ended := strings.Cut(rest, "      # Versions are compared")

	if !found || !ended {
		t.Fatal("values.yaml has no list of Linkerd categories")
	}

	var categories []string

	for line := range strings.Lines(list) {
		text := strings.TrimPrefix(line, "      #   ")
		if text == line || strings.HasPrefix(text, " ") {
			continue // a continuation line
		}

		categories = append(categories, strings.Fields(text)[0])
	}

	return categories
}

// TestLinkerdCategories checks that the categories that values.yaml documents and enables
// are categories of the Linkerd library that the Checker uses.
func TestLinkerdCategories(t *testing.T) {
	known := map[string]bool{}
	for _, id := range []healthcheck.CategoryID{
		healthcheck.KubernetesAPIChecks,
		healthcheck.KubernetesVersionChecks,
		healthcheck.LinkerdVersionChecks,
		healthcheck.LinkerdControlPlaneExistenceChecks,
		healthcheck.LinkerdConfigChecks,
		healthcheck.LinkerdIdentity,
		healthcheck.LinkerdWebhooksAndAPISvcTLS,
		healthcheck.LinkerdControlPlaneVersionChecks,
		healthcheck.LinkerdControlPlaneProxyChecks,
		healthcheck.LinkerdHAChecks,
		healthcheck.LinkerdDataPlaneChecks,
		healthcheck.LinkerdIdentityDataPlane,
		healthcheck.LinkerdCNIPluginChecks,
		healthcheck.LinkerdExtensionChecks,
	} {
		known[string(id)] = true
	}

	documented := documentedCategories(t)
	if len(documented) != len(known) {
		t.Errorf("values.yaml documents %d categories, want %d: %v", len(documented), len(known), documented)
	}

	for _, category := range documented {
		if !known[category] {
			t.Errorf("values.yaml documents the unknown category %q", category)
		}
	}

	for _, category := range loadValues(t).Healthcheck.Linkerd.Checks {
		if !known[category] {
			t.Errorf("the default checks contain the unknown category %q", category)
		}
	}
}

// waitForRun starts the checks of the health check endpoint the way the Checker does and
// returns the HTTP status and answer of the endpoint after the first run.
func waitForRun(t *testing.T, cfg *config.Healthcheck) (int, map[string]any) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	cache := checks.NewCachedResponses(ctx, cfg)

	deadline := time.Now().Add(15 * time.Second)
	for cache.Response() == nil {
		if time.Now().After(deadline) {
			t.Fatal("the first run of the checks did not finish")
		}

		time.Sleep(20 * time.Millisecond)
	}

	return cache.Status(), cache.Response()
}

// clusterResults returns the results of the `cluster` group of an answer.
func clusterResults(t *testing.T, answer map[string]any) []*checks.Response {
	t.Helper()

	results, ok := answer["cluster"].([]*checks.Response)
	if !ok || len(results) != 1 {
		t.Fatalf("want one result in the cluster group, got %#v", answer)
	}

	return results
}

// TestHealthcheckExamples runs the example checks of values.yaml with the Checker's check code
// against a test server, and checks the answers that values.yaml describes.
func TestHealthcheckExamples(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status": "UP"} ok, all checks passed`))
	}))
	t.Cleanup(healthy.Close)

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failing.Close)

	// group renders a values file with the example checks for one resource of the cluster
	// group and loads the configuration.
	group := func(t *testing.T, url, retry string) *config.Healthcheck {
		t.Helper()

		values := "config:\n  healthcheck:\n    kubernetes: { enabled: false }\n" +
			"    linkerd: { enabled: false }\n    cluster:\n      enabled: true\n      resources:\n" +
			"        - name: example\n          url: " + url + "\n" + retry +
			indent(valuesComment(t, "Example checks:"), "          ")

		return &loadValues(t, "-f", writeValues(t, values)).Healthcheck
	}

	t.Run("all example checks pass", func(t *testing.T) {
		status, answer := waitForRun(t, group(t, healthy.URL, ""))
		result := clusterResults(t, answer)[0]

		if status != http.StatusOK || result.Status != checks.OK || len(result.Errors) != 0 {
			t.Errorf("got HTTP %d and %+v", status, result)
		}

		if result.Name != "example" || result.URL != healthy.URL {
			t.Errorf("name and url: got %q and %q", result.Name, result.URL)
		}
	})

	t.Run("a failing service", func(t *testing.T) {
		status, answer := waitForRun(t, group(t, failing.URL, ""))
		result := clusterResults(t, answer)[0]

		if status != http.StatusServiceUnavailable || result.Status != checks.NOTOK || len(result.Errors) == 0 {
			t.Errorf("got HTTP %d and %+v", status, result)
		}
	})

	t.Run("a failure that retry.maxRetries tolerates", func(t *testing.T) {
		status, answer := waitForRun(t, group(t, failing.URL, "          retry: { maxRetries: 2 }\n"))
		result := clusterResults(t, answer)[0]

		if status != http.StatusOK || result.Status != "OK [FAILURE TOLERATED ON RETRY]" {
			t.Errorf("got HTTP %d and %+v", status, result)
		}
	})

	t.Run("a resource without checks passes with any answer", func(t *testing.T) {
		cfg := group(t, failing.URL, "")
		cfg.Cluster.Resources[0].Checks = nil

		status, _ := waitForRun(t, cfg)
		if status != http.StatusOK {
			t.Errorf("got HTTP %d", status)
		}
	})
}

// indent prefixes every line of text with prefix.
func indent(text, prefix string) string {
	var out strings.Builder
	for line := range strings.Lines(text) {
		out.WriteString(prefix + line)
	}

	return out.String()
}
