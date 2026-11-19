package ai_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hackclub/hackatime-cli/pkg/ai"
	"github.com/hackclub/hackatime-cli/pkg/heartbeat"
	"github.com/hackclub/hackatime-cli/pkg/ini"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// humanEntity returns a file next to the AI-edited entity, so both belong to
// the same project.
func humanEntity(aiEntity string, name string) string {
	return filepath.ToSlash(filepath.Join(filepath.Dir(aiEntity), name))
}

// newGitProject creates a folder that project detection treats as a project
// root and returns it with forward slashes, as editor plugins send it.
func newGitProject(t *testing.T, parent string, name string) string {
	t.Helper()

	dir := filepath.Join(parent, name)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

	return filepath.ToSlash(dir)
}

// aiSyncHandle runs WithAISync over one Claude Code transcript that edits
// aiEntity at 2026-03-18T12:00:00Z (1773835200).
func aiSyncHandle(t *testing.T, aiEntity string) heartbeat.Handle {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	transcriptDir := filepath.Join(home, ".claude", "projects", "sample-project")
	require.NoError(t, os.MkdirAll(transcriptDir, 0o755))

	transcript := "{\"timestamp\":\"2026-03-18T12:00:00Z\",\"version\":\"2.1.45\"," +
		"\"toolUseResult\":{\"filePath\":\"" + aiEntity + "\"," +
		"\"structuredPatch\":[{\"oldLines\":3,\"newLines\":5}]}}\n"
	require.NoError(t, os.WriteFile(filepath.Join(transcriptDir, "session.jsonl"), []byte(transcript), 0o644))

	internal, err := os.CreateTemp(t.TempDir(), "wakatime-internal")
	require.NoError(t, err)
	require.NoError(t, internal.Close())

	v := viper.New()
	v.Set("internal-config", internal.Name())
	v.Set("internal.ai_logs_last_parsed_at", time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC).Format(ini.DateFormat))

	return ai.WithAISync(ai.Config{
		Plugin: "plugin/0.0.1",
		V:      v,
	})(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		results := make([]heartbeat.Result, len(hh))
		for i := range hh {
			results[i] = heartbeat.Result{Heartbeat: hh[i]}
		}

		return results, nil
	})
}

func categoriesByEntity(results []heartbeat.Result) map[string]string {
	categories := make(map[string]string, len(results))
	for _, result := range results {
		categories[result.Heartbeat.Entity] = result.Heartbeat.Category
	}

	return categories
}

func TestWithAISyncOnlyMarksHumanHeartbeatsInTheAIProject(t *testing.T) {
	root := t.TempDir()
	aiProject := newGitProject(t, root, "course-2")
	humanProject := newGitProject(t, root, "rogueshot-2")

	aiEntity := aiProject + "/ai-assignment.py"
	require.NoError(t, os.WriteFile(filepath.FromSlash(aiEntity), []byte("print(1)\n"), 0o644))

	handle := aiSyncHandle(t, aiEntity)

	// The reported case: an agent works in one project while the developer
	// edits a game in another, with a plugin that never reports human line
	// changes.
	otherProjectSoon := humanProject + "/scripts/weapon/weapon.gd"
	otherProjectLater := humanProject + "/scripts/items/inventory.gd"
	sameProjectLater := aiProject + "/notes.md"
	sameProjectNested := aiProject + "/src/deep/helper.py"

	got, err := handle(t.Context(), []heartbeat.Heartbeat{
		{
			Entity:     otherProjectSoon,
			EntityType: heartbeat.FileType,
			Category:   "coding",
			Time:       1773835218,
			UserAgent:  "godot/4.7.2",
		},
		{
			Entity:     otherProjectLater,
			EntityType: heartbeat.FileType,
			Category:   "coding",
			Time:       1773836700,
			UserAgent:  "godot/4.7.2",
		},
		{
			Entity:     sameProjectLater,
			EntityType: heartbeat.FileType,
			Category:   "coding",
			Time:       1773836700,
			UserAgent:  "editor/1.2.3",
		},
		{
			Entity:     sameProjectNested,
			EntityType: heartbeat.FileType,
			Category:   "coding",
			Time:       1773835260,
			UserAgent:  "editor/1.2.3",
		},
	})
	require.NoError(t, err)

	categories := categoriesByEntity(got)

	assert.Equal(t, "coding", categories[otherProjectSoon])
	assert.Equal(t, "coding", categories[otherProjectLater])
	assert.Equal(t, "ai coding", categories[sameProjectLater])
	assert.Equal(t, "ai coding", categories[sameProjectNested])
	assert.Equal(t, "ai coding", categories[aiEntity])
}

func TestWithAISyncUsesProjectFolderOverrideForScope(t *testing.T) {
	root := t.TempDir()
	aiProject := newGitProject(t, root, "course-2")
	humanProject := newGitProject(t, root, "rogueshot-2")

	aiEntity := aiProject + "/ai-assignment.py"
	require.NoError(t, os.WriteFile(filepath.FromSlash(aiEntity), []byte("print(1)\n"), 0o644))

	handle := aiSyncHandle(t, aiEntity)

	// Plugins can name the project folder explicitly. A file outside it still
	// belongs to the named project.
	inNamedProject := root + "/elsewhere/scratch.py"
	inOtherProject := humanProject + "/main.gd"

	got, err := handle(t.Context(), []heartbeat.Heartbeat{
		{
			Entity:              inNamedProject,
			EntityType:          heartbeat.FileType,
			Category:            "coding",
			ProjectPathOverride: aiProject,
			Time:                1773835260,
			UserAgent:           "editor/1.2.3",
		},
		{
			Entity:              inOtherProject,
			EntityType:          heartbeat.FileType,
			Category:            "coding",
			ProjectPathOverride: humanProject,
			Time:                1773835260,
			UserAgent:           "editor/1.2.3",
		},
	})
	require.NoError(t, err)

	categories := categoriesByEntity(got)

	assert.Equal(t, "ai coding", categories[inNamedProject])
	assert.Equal(t, "coding", categories[inOtherProject])
}

func TestWithAISyncHumanEditInOtherProjectDoesNotEndAIWindow(t *testing.T) {
	root := t.TempDir()
	aiProject := newGitProject(t, root, "course-2")
	humanProject := newGitProject(t, root, "rogueshot-2")

	aiEntity := aiProject + "/ai-assignment.py"
	require.NoError(t, os.WriteFile(filepath.FromSlash(aiEntity), []byte("print(1)\n"), 0o644))

	handle := aiSyncHandle(t, aiEntity)

	otherProjectEdit := humanProject + "/main.gd"
	sameProjectAfterOtherEdit := aiProject + "/review.py"

	got, err := handle(t.Context(), []heartbeat.Heartbeat{
		{
			Entity:           otherProjectEdit,
			EntityType:       heartbeat.FileType,
			Category:         "coding",
			HumanLineChanges: heartbeat.PointerTo(4),
			Time:             1773835500,
			UserAgent:        "editor/1.2.3",
		},
		{
			Entity:           sameProjectAfterOtherEdit,
			EntityType:       heartbeat.FileType,
			Category:         "coding",
			HumanLineChanges: heartbeat.PointerTo(0),
			Time:             1773836000,
			UserAgent:        "editor/1.2.3",
		},
	})
	require.NoError(t, err)

	categories := categoriesByEntity(got)

	assert.Equal(t, "coding", categories[otherProjectEdit])
	assert.Equal(t, "ai coding", categories[sameProjectAfterOtherEdit])
}

func TestWithAISyncDoesNotMarkHumanHeartbeatsWithoutAFolder(t *testing.T) {
	root := t.TempDir()
	aiProject := newGitProject(t, root, "course-2")

	aiEntity := aiProject + "/ai-assignment.py"
	require.NoError(t, os.WriteFile(filepath.FromSlash(aiEntity), []byte("print(1)\n"), 0o644))

	handle := aiSyncHandle(t, aiEntity)

	app := "Godot Engine"

	got, err := handle(t.Context(), []heartbeat.Heartbeat{
		{
			Entity:     app,
			EntityType: heartbeat.AppType,
			Category:   "coding",
			Time:       1773835260,
			UserAgent:  "godot/4.7.2",
		},
	})
	require.NoError(t, err)

	categories := categoriesByEntity(got)

	assert.Equal(t, "coding", categories[app])
}
