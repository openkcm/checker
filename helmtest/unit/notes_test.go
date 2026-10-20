//go:build helmtest

package unit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// releaseNotes renders templates/NOTES.txt with the arguments. `helm template` does not
// render the release notes and `helm install --dry-run` needs a cluster with Helm 3, so the
// test renders a copy of the chart in which the notes are a ConfigMap.
func releaseNotes(t *testing.T, args ...string) string {
	t.Helper()

	chart := filepath.Join(t.TempDir(), "checker")

	err := os.CopyFS(chart, os.DirFS(chartPath))
	if err != nil {
		t.Fatal(err)
	}

	notesPath := filepath.Join(chart, "templates", "NOTES.txt")

	notes, err := os.ReadFile(notesPath)
	if err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"_notes_under_test.tpl": "{{- define \"checker.notesUnderTest\" -}}\n" + string(notes) + "\n{{- end -}}\n",
		"notes-under-test.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: notes-under-test\n" +
			"data:\n  notes: {{ include \"checker.notesUnderTest\" . | toJson }}\n",
	}
	for name, content := range files {
		err = os.WriteFile(filepath.Join(chart, "templates", name), []byte(content), 0o600)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = os.Remove(notesPath)
	if err != nil {
		t.Fatal(err)
	}

	out, err := runHelm(t, append([]string{"template", releaseName, chart,
		"--show-only", "templates/notes-under-test.yaml"}, args...)...)
	if err != nil {
		t.Fatalf("helm template failed: %v", err)
	}

	var cm struct {
		Data struct {
			Notes string `yaml:"notes"`
		} `yaml:"data"`
	}

	err = yaml.Unmarshal([]byte(out), &cm)
	if err != nil {
		t.Fatal(err)
	}

	return cm.Data.Notes
}

// TestReleaseNotes checks the endpoints and the warnings of the release notes.
func TestReleaseNotes(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		notes := releaseNotes(t)

		for _, want := range []string{
			"http://checker.services.svc:8080/healthz",
			"ConfigMap checker-config",
			"port-forward service/checker 8080:http",
		} {
			if !strings.Contains(notes, want) {
				t.Errorf("the notes do not contain %q:\n%s", want, notes)
			}
		}

		if strings.Contains(notes, "WARNINGS") {
			t.Errorf("warnings with the default values:\n%s", notes)
		}
	})

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "two endpoints with the same path",
			args: []string{"--set-json", `config.healthchecks=[{"enabled":true}]`},
			want: "do not all have their own path (/healthz, /healthz)",
		},
		{
			name: "versions on the path of the health check",
			args: []string{"--set", "config.versions.enabled=true", "--set", "config.versions.endpoint=/healthz"},
			want: "do not all have their own path",
		},
		{
			name: "no token",
			args: []string{"--set", "serviceAccount.automount=false"},
			want: "serviceAccount.automount is false",
		},
		{
			name: "no token, linkerd check in healthchecks",
			args: []string{
				"--set", "serviceAccount.automount=false", "--set", "config.healthcheck.enabled=false",
				"--set-json", `config.healthchecks=[{"enabled":true,"linkerd":{"enabled":true}}]`,
			},
			want: "serviceAccount.automount is false",
		},
		{
			name: "no RBAC",
			args: []string{"--set", "rbac.create=false"},
			want: "rbac.create is false",
		},
		{
			name: "status server off",
			args: []string{"--set", "config.status.enabled=false"},
			want: "config.status.enabled is false",
		},
		{
			name: "profiling",
			args: []string{"--set", "config.status.profiling=true"},
			want: "config.status.profiling is true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes := releaseNotes(t, tt.args...)
			if !strings.Contains(notes, tt.want) {
				t.Errorf("the notes do not contain %q:\n%s", tt.want, notes)
			}
		})
	}

	t.Run("no warnings without API checks", func(t *testing.T) {
		notes := releaseNotes(t, "--set", "serviceAccount.automount=false", "--set", "rbac.create=false",
			"--set", "config.healthcheck.kubernetes.enabled=false", "--set", "config.healthcheck.linkerd.enabled=false")
		if strings.Contains(notes, "WARNINGS") {
			t.Errorf("warnings:\n%s", notes)
		}
	})
}
