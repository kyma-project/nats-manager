package rbac_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// clusterRole represents the relevant fields of a k8s ClusterRole for testing.
type clusterRole struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name   string            `yaml:"name"`
		Labels map[string]string `yaml:"labels"`
	} `yaml:"metadata"`
	Rules []struct {
		APIGroups []string `yaml:"apiGroups"`
		Resources []string `yaml:"resources"`
		Verbs     []string `yaml:"verbs"`
	} `yaml:"rules"`
}

func loadClusterRole(t *testing.T, filename string) clusterRole {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Dir(currentFile)

	data, err := os.ReadFile(filepath.Join(dir, "..", "customer_rbac", filename))
	require.NoError(t, err)

	var role clusterRole
	require.NoError(t, yaml.Unmarshal(data, &role))
	return role
}

func TestViewRoleStructure(t *testing.T) {
	role := loadClusterRole(t, "kyma_nats_view_role.yaml")

	// Verify basic metadata.
	assert.Equal(t, "rbac.authorization.k8s.io/v1", role.APIVersion)
	assert.Equal(t, "ClusterRole", role.Kind)
	assert.Equal(t, "kyma-nats-view", role.Metadata.Name)

	// Verify aggregation label.
	assert.Equal(t, "true", role.Metadata.Labels["rbac.authorization.k8s.io/aggregate-to-view"])
	// Must NOT aggregate to edit.
	_, hasEdit := role.Metadata.Labels["rbac.authorization.k8s.io/aggregate-to-edit"]
	assert.False(t, hasEdit, "view role must not have aggregate-to-edit label")

	// Verify rules.
	require.Len(t, role.Rules, 2)

	// Rule 1: nats resource.
	assert.Equal(t, []string{"operator.kyma-project.io"}, role.Rules[0].APIGroups)
	assert.Equal(t, []string{"nats"}, role.Rules[0].Resources)
	assert.ElementsMatch(t, []string{"get", "list", "watch"}, role.Rules[0].Verbs)
	// View must NOT have write verbs.
	for _, verb := range role.Rules[0].Verbs {
		assert.NotContains(t, []string{"create", "update", "patch", "delete", "deletecollection"}, verb,
			"view role must not have write verbs on nats resource")
	}

	// Rule 2: nats/status subresource.
	assert.Equal(t, []string{"operator.kyma-project.io"}, role.Rules[1].APIGroups)
	assert.Equal(t, []string{"nats/status"}, role.Rules[1].Resources)
	assert.Equal(t, []string{"get"}, role.Rules[1].Verbs)
}

func TestEditRoleStructure(t *testing.T) {
	role := loadClusterRole(t, "kyma_nats_edit_role.yaml")

	// Verify basic metadata.
	assert.Equal(t, "rbac.authorization.k8s.io/v1", role.APIVersion)
	assert.Equal(t, "ClusterRole", role.Kind)
	assert.Equal(t, "kyma-nats-edit", role.Metadata.Name)

	// Verify aggregation label.
	assert.Equal(t, "true", role.Metadata.Labels["rbac.authorization.k8s.io/aggregate-to-edit"])
	// Must NOT aggregate to view (edit is a superset, not aggregated into view).
	_, hasView := role.Metadata.Labels["rbac.authorization.k8s.io/aggregate-to-view"]
	assert.False(t, hasView, "edit role must not have aggregate-to-view label")

	// Verify rules.
	require.Len(t, role.Rules, 2)

	// Rule 1: nats resource — must have full CRUD.
	assert.Equal(t, []string{"operator.kyma-project.io"}, role.Rules[0].APIGroups)
	assert.Equal(t, []string{"nats"}, role.Rules[0].Resources)
	assert.ElementsMatch(t,
		[]string{"get", "list", "watch", "create", "update", "patch", "delete", "deletecollection"},
		role.Rules[0].Verbs,
	)

	// Rule 2: nats/status subresource — read only (controller owns writes).
	assert.Equal(t, []string{"operator.kyma-project.io"}, role.Rules[1].APIGroups)
	assert.Equal(t, []string{"nats/status"}, role.Rules[1].Resources)
	assert.Equal(t, []string{"get"}, role.Rules[1].Verbs)
}

func TestEditRoleIsSupersetOfViewRole(t *testing.T) {
	viewRole := loadClusterRole(t, "kyma_nats_view_role.yaml")
	editRole := loadClusterRole(t, "kyma_nats_edit_role.yaml")

	// Every verb in view must also be in edit for the same resource.
	for _, viewRule := range viewRole.Rules {
		found := false
		for _, editRule := range editRole.Rules {
			if viewRule.Resources[0] == editRule.Resources[0] {
				found = true
				for _, verb := range viewRule.Verbs {
					assert.Contains(t, editRule.Verbs, verb,
						"edit role must contain all verbs from view role for resource %s", viewRule.Resources[0])
				}
			}
		}
		assert.True(t, found, "edit role must cover resource %s from view role", viewRule.Resources[0])
	}
}

func TestRolesDoNotExposeSecrets(t *testing.T) {
	for _, filename := range []string{"kyma_nats_view_role.yaml", "kyma_nats_edit_role.yaml"} {
		role := loadClusterRole(t, filename)
		for _, rule := range role.Rules {
			for _, resource := range rule.Resources {
				assert.NotEqual(t, "secrets", resource,
					"customer-facing role %s must not expose secrets", role.Metadata.Name)
			}
		}
	}
}

func TestRolesOnlyTargetNATSAPIGroup(t *testing.T) {
	for _, filename := range []string{"kyma_nats_view_role.yaml", "kyma_nats_edit_role.yaml"} {
		role := loadClusterRole(t, filename)
		for _, rule := range role.Rules {
			for _, apiGroup := range rule.APIGroups {
				assert.Equal(t, "operator.kyma-project.io", apiGroup,
					"customer-facing role %s must only target the NATS API group", role.Metadata.Name)
			}
		}
	}
}
