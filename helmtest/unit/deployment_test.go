//go:build helmtest

package unit_test

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"testing"
)

// deployment is the part of the Deployment that the tests check.
type deployment struct {
	Spec struct {
		Replicas *int `yaml:"replicas"`
		Template struct {
			Metadata struct {
				Annotations map[string]string `yaml:"annotations"`
			} `yaml:"metadata"`
			Spec struct {
				ServiceAccountName           string `yaml:"serviceAccountName"`
				AutomountServiceAccountToken *bool  `yaml:"automountServiceAccountToken"`
				EnableServiceLinks           *bool  `yaml:"enableServiceLinks"`
				SecurityContext              struct {
					RunAsNonRoot   bool  `yaml:"runAsNonRoot"`
					RunAsUser      int64 `yaml:"runAsUser"`
					RunAsGroup     int64 `yaml:"runAsGroup"`
					FSGroup        int64 `yaml:"fsGroup"`
					SeccompProfile struct {
						Type string `yaml:"type"`
					} `yaml:"seccompProfile"`
				} `yaml:"securityContext"`
				Containers []container `yaml:"containers"`
				Volumes    []struct {
					Name      string `yaml:"name"`
					ConfigMap *struct {
						Name string `yaml:"name"`
					} `yaml:"configMap"`
					EmptyDir *struct {
						SizeLimit string `yaml:"sizeLimit"`
					} `yaml:"emptyDir"`
				} `yaml:"volumes"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

type container struct {
	Name            string   `yaml:"name"`
	Image           string   `yaml:"image"`
	Command         []string `yaml:"command"`
	Args            []string `yaml:"args"`
	SecurityContext struct {
		AllowPrivilegeEscalation *bool `yaml:"allowPrivilegeEscalation"`
		ReadOnlyRootFilesystem   bool  `yaml:"readOnlyRootFilesystem"`
		Capabilities             struct {
			Drop []string `yaml:"drop"`
		} `yaml:"capabilities"`
	} `yaml:"securityContext"`
	Ports []struct {
		Name          string `yaml:"name"`
		ContainerPort int    `yaml:"containerPort"`
	} `yaml:"ports"`
	StartupProbe   *probe `yaml:"startupProbe"`
	LivenessProbe  *probe `yaml:"livenessProbe"`
	ReadinessProbe *probe `yaml:"readinessProbe"`
	Resources      struct {
		Requests map[string]string `yaml:"requests"`
		Limits   map[string]string `yaml:"limits"`
	} `yaml:"resources"`
	VolumeMounts []struct {
		Name      string `yaml:"name"`
		MountPath string `yaml:"mountPath"`
		SubPath   string `yaml:"subPath"`
		ReadOnly  bool   `yaml:"readOnly"`
	} `yaml:"volumeMounts"`
}

type probe struct {
	HTTPGet struct {
		Path string `yaml:"path"`
		Port string `yaml:"port"`
	} `yaml:"httpGet"`
}

// checkerContainer renders the chart and returns its Deployment and the Checker container.
func checkerContainer(t *testing.T, args ...string) (deployment, container) {
	t.Helper()

	d := object[deployment](t, render(t, args...), "Deployment")

	for _, c := range d.Spec.Template.Spec.Containers {
		if c.Name == "checker" {
			return d, c
		}
	}

	t.Fatal("the Deployment has no container checker")

	return d, container{}
}

// TestImage checks the image reference: the tag defaults to the chart's appVersion.
func TestImage(t *testing.T) {
	const repository = "ghcr.io/openkcm/checker/images/checker"

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "default", want: repository + ":" + appVersion(t)},
		{name: "tag", args: []string{"--set", "image.tag=v0.3.1"}, want: repository + ":v0.3.1"},
		{
			name: "digest before tag",
			args: []string{"--set", "image.tag=v0.3.1", "--set", "image.digest=sha256:0e07"},
			want: repository + "@sha256:0e07",
		},
		{
			name: "full name in repository",
			args: []string{"--set", "image.registry=", "--set", "image.repository=localhost/checker"},
			want: "localhost/checker:" + appVersion(t),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, c := checkerContainer(t, tt.args...)
			if c.Image != tt.want {
				t.Errorf("image: got %q, want %q", c.Image, tt.want)
			}
		})
	}
}

// TestCommand checks that the image's command runs by default, and both forms of the
// override.
func TestCommand(t *testing.T) {
	_, c := checkerContainer(t)
	if c.Command != nil || !slices.Equal(c.Args, []string{"--graceful-shutdown=5"}) {
		t.Errorf("default: got command %v and args %v", c.Command, c.Args)
	}

	_, c = checkerContainer(t, "--set", "image.command=/checker")
	if !slices.Equal(c.Command, []string{"/checker"}) {
		t.Errorf("string: got command %v", c.Command)
	}

	_, c = checkerContainer(t, "--set-json", `image.command=["/checker","--version"]`)
	if !slices.Equal(c.Command, []string{"/checker", "--version"}) {
		t.Errorf("list: got command %v", c.Command)
	}
}

// TestSecurityDefaults checks the security settings that values.yaml and the README
// describe.
func TestSecurityDefaults(t *testing.T) {
	d, c := checkerContainer(t)
	pod := d.Spec.Template.Spec

	sc := pod.SecurityContext
	if !sc.RunAsNonRoot || sc.RunAsUser != 65532 || sc.RunAsGroup != 65532 || sc.FSGroup != 65532 ||
		sc.SeccompProfile.Type != "RuntimeDefault" {
		t.Errorf("pod security context: got %+v", sc)
	}

	csc := c.SecurityContext
	if csc.AllowPrivilegeEscalation == nil || *csc.AllowPrivilegeEscalation || !csc.ReadOnlyRootFilesystem ||
		!slices.Equal(csc.Capabilities.Drop, []string{"ALL"}) {
		t.Errorf("container security context: got %+v", csc)
	}

	if pod.EnableServiceLinks == nil || *pod.EnableServiceLinks {
		t.Error("enableServiceLinks is not false")
	}

	// The kubernetes and linkerd checks need the token.
	if pod.AutomountServiceAccountToken == nil || !*pod.AutomountServiceAccountToken {
		t.Error("automountServiceAccountToken is not true")
	}

	d, _ = checkerContainer(t, "--set", "serviceAccount.automount=false")
	if token := d.Spec.Template.Spec.AutomountServiceAccountToken; token == nil || *token {
		t.Error("serviceAccount.automount=false does not switch the token off")
	}

	if pod.ServiceAccountName != "checker" {
		t.Errorf("serviceAccountName: got %q", pod.ServiceAccountName)
	}

	if c.Resources.Requests["cpu"] == "" || c.Resources.Requests["memory"] == "" ||
		c.Resources.Limits["memory"] == "" || c.Resources.Limits["ephemeral-storage"] == "" {
		t.Errorf("resources: got %+v", c.Resources)
	}
}

// TestVolumes checks that the configuration is mounted as /etc/checker/config.yaml and that
// /tmp is writable.
func TestVolumes(t *testing.T) {
	d, c := checkerContainer(t)

	mounts := map[string]string{}

	for _, m := range c.VolumeMounts {
		mounts[m.Name] = m.MountPath

		if m.Name == "config" && (m.SubPath != "config.yaml" || !m.ReadOnly) {
			t.Errorf("config mount: got %+v", m)
		}
	}

	if mounts["config"] != "/etc/checker/config.yaml" || mounts["tmp"] != "/tmp" {
		t.Errorf("mounts: got %v", mounts)
	}

	for _, v := range d.Spec.Template.Spec.Volumes {
		switch v.Name {
		case "config":
			if v.ConfigMap == nil || v.ConfigMap.Name != "checker-config" {
				t.Errorf("config volume: got %+v", v.ConfigMap)
			}
		case "tmp":
			if v.EmptyDir == nil || v.EmptyDir.SizeLimit != "64Mi" {
				t.Errorf("tmp volume: got %+v", v.EmptyDir)
			}
		}
	}

	_, c = checkerContainer(t, "--set", "tmpVolume.enabled=false")
	for _, m := range c.VolumeMounts {
		if m.Name == "tmp" {
			t.Error("tmpVolume.enabled=false still mounts /tmp")
		}
	}
}

// TestProbes checks that the probes use the status port and its probe endpoints.
func TestProbes(t *testing.T) {
	_, c := checkerContainer(t)

	for name, p := range map[string]*probe{
		"startup": c.StartupProbe, "liveness": c.LivenessProbe, "readiness": c.ReadinessProbe,
	} {
		if p == nil || p.HTTPGet.Port != "http-status" || !strings.HasPrefix(p.HTTPGet.Path, "/probe/") {
			t.Errorf("%s probe: got %+v", name, p)
		}
	}
}

// TestChecksumConfig checks that the checksum annotation is the hash of config.yaml, so that
// a configuration change restarts the pods.
func TestChecksumConfig(t *testing.T) {
	for _, args := range [][]string{nil, {"--set", "config.environment=staging"}} {
		out := render(t, args...)

		sum := sha256.Sum256([]byte(object[configMap](t, out, "ConfigMap").Data["config.yaml"]))
		got := object[deployment](t, out, "Deployment").Spec.Template.Metadata.Annotations["checksum/config"]

		if got != hex.EncodeToString(sum[:]) {
			t.Errorf("%v: checksum/config is not the hash of config.yaml", args)
		}
	}
}

// TestConfigMapName checks the mutable and the immutable ConfigMap.
func TestConfigMapName(t *testing.T) {
	cm := object[configMap](t, render(t), "ConfigMap")
	if cm.Metadata.Name != "checker-config" || cm.Immutable {
		t.Errorf("default: got %q, immutable %v", cm.Metadata.Name, cm.Immutable)
	}

	for _, flag := range []string{"configMap.immutable=true", "config.isImmutable=true"} {
		out := render(t, "--set", flag)

		cm := object[configMap](t, out, "ConfigMap")
		volumes := object[deployment](t, out, "Deployment").Spec.Template.Spec.Volumes

		if !cm.Immutable || !strings.HasPrefix(cm.Metadata.Name, "checker-config-") ||
			volumes[0].ConfigMap == nil || volumes[0].ConfigMap.Name != cm.Metadata.Name {
			t.Errorf("%s: got %q, immutable %v", flag, cm.Metadata.Name, cm.Immutable)
		}
	}
}

// TestReplicasAndAutoscaler checks that the autoscaler replaces replicaCount, and the checks
// of the autoscaler and the disruption budget.
func TestReplicasAndAutoscaler(t *testing.T) {
	d := object[deployment](t, render(t, "--set", "replicaCount=2"), "Deployment")
	if d.Spec.Replicas == nil || *d.Spec.Replicas != 2 {
		t.Errorf("replicas: got %v", d.Spec.Replicas)
	}

	out := render(t, "--set", "hpa.enabled=true")
	if d := object[deployment](t, out, "Deployment"); d.Spec.Replicas != nil {
		t.Error("the Deployment sets replicas with the autoscaler")
	}

	object[map[string]any](t, out, "HorizontalPodAutoscaler")

	err := renderError(t, "--set", "hpa.enabled=true", "--set", "hpa.targetCPUUtilizationPercentage=null",
		"--set", "hpa.targetMemoryUtilizationPercentage=null")
	if !strings.Contains(err.Error(), "hpa.enabled requires") {
		t.Errorf("hpa without metrics: got %v", err)
	}

	err = renderError(t, "--set", "pod.disruptionBudget.enabled=true",
		"--set", "pod.disruptionBudget.minAvailable=1", "--set", "pod.disruptionBudget.maxUnavailable=1")
	if !strings.Contains(err.Error(), "not both") {
		t.Errorf("disruption budget with both values: got %v", err)
	}
}

// TestNetworkPolicy checks the rules of the network policy.
func TestNetworkPolicy(t *testing.T) {
	if objs := objects[map[string]any](t, render(t), "NetworkPolicy"); len(objs) != 0 {
		t.Error("the network policy is created by default")
	}

	type policy struct {
		Spec struct {
			Ingress []struct {
				Ports []struct {
					Port string `yaml:"port"`
				} `yaml:"ports"`
				From []map[string]any `yaml:"from"`
			} `yaml:"ingress"`
		} `yaml:"spec"`
	}

	np := object[policy](t, render(t, "--set", "networkPolicy.enabled=true",
		"--set-json", `networkPolicy.http.from=[{"namespaceSelector":{}}]`), "NetworkPolicy")

	if len(np.Spec.Ingress) != 1 || np.Spec.Ingress[0].Ports[0].Port != "http" {
		t.Errorf("ingress: got %+v", np.Spec.Ingress)
	}
}
