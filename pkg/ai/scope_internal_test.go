package ai

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hackclub/hackatime-cli/pkg/heartbeat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityScopeRelated(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "work")
	project := activityScope{path: filepath.Join(root, "project"), anchored: true}

	tests := map[string]struct {
		Other    activityScope
		Expected bool
	}{
		"same folder": {
			Other:    activityScope{path: filepath.Join(root, "project")},
			Expected: true,
		},
		"nested folder": {
			Other:    activityScope{path: filepath.Join(root, "project", "src")},
			Expected: true,
		},
		"sibling project": {
			Other:    activityScope{path: filepath.Join(root, "project-two"), anchored: true},
			Expected: false,
		},
		"unanchored parent": {
			Other:    activityScope{path: root},
			Expected: false,
		},
		"anchored parent": {
			Other:    activityScope{path: root, anchored: true},
			Expected: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.Expected, project.related(test.Other))
			assert.Equal(t, test.Expected, test.Other.related(project))
		})
	}
}

func TestScopeResolverFindsProjectRoot(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	require.NoError(t, os.MkdirAll(filepath.Join(project, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(project, "src", "pkg"), 0o755))

	resolver := newScopeResolver()

	scope, ok := resolver.scope(heartbeat.Heartbeat{
		Entity:     filepath.ToSlash(filepath.Join(project, "src", "pkg", "main.go")),
		EntityType: heartbeat.FileType,
	})
	require.True(t, ok)
	assert.Equal(t, activityScope{path: normalizeScopePath(project), anchored: true}, scope)

	loose := filepath.Join(root, "loose")
	scope, ok = resolver.scope(heartbeat.Heartbeat{
		Entity:     filepath.Join(loose, "notes.txt"),
		EntityType: heartbeat.FileType,
	})
	require.True(t, ok)
	assert.Equal(t, activityScope{path: normalizeScopePath(loose)}, scope)
}

func TestScopeResolverWithoutFolder(t *testing.T) {
	resolver := newScopeResolver()

	_, ok := resolver.scope(heartbeat.Heartbeat{Entity: "Copilot session", EntityType: heartbeat.AppType})
	assert.False(t, ok)

	_, ok = resolver.scope(heartbeat.Heartbeat{Entity: "relative/main.go", EntityType: heartbeat.FileType})
	assert.False(t, ok)

	scope, ok := resolver.scope(heartbeat.Heartbeat{
		Entity:              "Copilot session",
		EntityType:          heartbeat.AppType,
		ProjectPathOverride: "/work/project",
	})
	require.True(t, ok)
	assert.True(t, scope.anchored)
}
