//go:build helmtest

package unit_test

import (
	"slices"
	"testing"
)

type clusterRole struct {
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Rules []struct {
		APIGroups     []string `yaml:"apiGroups"`
		Resources     []string `yaml:"resources"`
		ResourceNames []string `yaml:"resourceNames"`
		Verbs         []string `yaml:"verbs"`
	} `yaml:"rules"`
}

type clusterRoleBinding struct {
	RoleRef struct {
		Kind string `yaml:"kind"`
		Name string `yaml:"name"`
	} `yaml:"roleRef"`
	Subjects []struct {
		Kind      string `yaml:"kind"`
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"subjects"`
}

// TestRBAC checks the ClusterRole of the Linkerd checks: read-only, and Secrets and
// ConfigMaps only by name.
func TestRBAC(t *testing.T) {
	out := render(t)
	role := object[clusterRole](t, out, "ClusterRole")
	binding := object[clusterRoleBinding](t, out, "ClusterRoleBinding")

	if role.Metadata.Name != "checker" || role.Metadata.Namespace != "" {
		t.Errorf("ClusterRole metadata: got %+v", role.Metadata)
	}

	if binding.RoleRef.Kind != "ClusterRole" || binding.RoleRef.Name != role.Metadata.Name ||
		len(binding.Subjects) != 1 || binding.Subjects[0].Kind != "ServiceAccount" ||
		binding.Subjects[0].Name != "checker" || binding.Subjects[0].Namespace != namespace {
		t.Errorf("ClusterRoleBinding: got %+v", binding)
	}

	readOnly := []string{"get", "list", "watch"}

	for _, rule := range role.Rules {
		for _, verb := range rule.Verbs {
			if !slices.Contains(readOnly, verb) {
				t.Errorf("rule %v grants %q", rule.Resources, verb)
			}
		}

		for _, resource := range rule.Resources {
			if resource != "secrets" && resource != "configmaps" {
				continue
			}

			if len(rule.ResourceNames) == 0 || !slices.Equal(rule.Verbs, []string{"get"}) {
				t.Errorf("%s: want get by name, got %v of %v", resource, rule.Verbs, rule.ResourceNames)
			}
		}
	}

	t.Run("extraRules", func(t *testing.T) {
		role := object[clusterRole](t, render(t, "--set-json",
			`rbac.extraRules=[{"apiGroups":["apiextensions.k8s.io"],"resources":["customresourcedefinitions"],"verbs":["get"]}]`),
			"ClusterRole")

		last := role.Rules[len(role.Rules)-1]
		if !slices.Equal(last.Resources, []string{"customresourcedefinitions"}) {
			t.Errorf("last rule: got %+v", last)
		}
	})

	t.Run("rbac.create false", func(t *testing.T) {
		out := render(t, "--set", "rbac.create=false")
		if len(objects[clusterRole](t, out, "ClusterRole"))+len(objects[clusterRoleBinding](t, out, "ClusterRoleBinding")) != 0 {
			t.Error("rbac.create=false still creates RBAC objects")
		}
	})
}
