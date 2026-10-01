package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// setGlobalHooks installs hooks in the global config for the test's duration.
func setGlobalHooks(t *testing.T, hooks map[string][]map[string]any) {
	t.Helper()
	orig := globalconfig.DdevGlobalConfig.Hooks
	t.Cleanup(func() { globalconfig.DdevGlobalConfig.Hooks = orig })
	globalconfig.DdevGlobalConfig.Hooks = hooks
}

func TestTasksForHookMerge(t *testing.T) {
	setGlobalHooks(t, map[string][]map[string]any{
		"post-start": {
			{"exec": "same"},
			{"exec": "global only"},
			{"exec-host": "host", "service": "db"},
		},
	})
	app := &DdevApp{Hooks: map[string][]YAMLTask{
		"post-start": {
			{"exec": "project first"},
			{"exec": "same"},
		},
	}}
	require.NoError(t, app.loadGlobalHooks())

	tasks := app.tasksForHook("post-start")
	var got []string
	for _, ht := range tasks {
		s, _ := ht.task["exec"].(string)
		if s == "" {
			s, _ = ht.task["exec-host"].(string)
		}
		if ht.global {
			s = "global:" + s
		}
		got = append(got, s)
	}
	// Global tasks first, and the global copy of "same" is dropped in favor of the project's.
	require.Equal(t, []string{"global:global only", "global:host", "project first", "same"}, got)

	app.SkipGlobalHooks = true
	require.Len(t, app.tasksForHook("post-start"), 2)
	require.Empty(t, app.globalHooksSummary())
}

func TestLoadGlobalHooksValidation(t *testing.T) {
	setGlobalHooks(t, map[string][]map[string]any{"post-nonsense": {{"exec": "x"}}})
	app := &DdevApp{}
	require.ErrorContains(t, app.loadGlobalHooks(), "invalid hook post-nonsense")

	setGlobalHooks(t, map[string][]map[string]any{"post-start": {{"bogus": "x"}}})
	require.ErrorContains(t, app.loadGlobalHooks(), "invalid task")
}

func TestGlobalHooksSummary(t *testing.T) {
	setGlobalHooks(t, map[string][]map[string]any{
		"pre-start":  {{"exec-host": "a"}},
		"post-start": {{"exec": "b"}, {"exec": "c"}},
	})
	app := &DdevApp{}
	require.NoError(t, app.loadGlobalHooks())
	require.Equal(t, "2 post-start, 1 pre-start", app.globalHooksSummary())
}

// TestGlobalHooksNotInProjectConfigYAML checks that global hooks are not part of what
// WriteConfig marshals into a project's config.yaml.
func TestGlobalHooksNotInProjectConfigYAML(t *testing.T) {
	setGlobalHooks(t, map[string][]map[string]any{"post-start": {{"exec": "from global"}}})

	app, err := NewApp(t.TempDir(), false)
	require.NoError(t, err)
	require.NotEmpty(t, app.GlobalHooks)
	out, err := yaml.Marshal(app)
	require.NoError(t, err)
	require.NotContains(t, string(out), "from global")
}

func TestProcessHooksGlobal(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("exec-host hooks need bash")
	}
	setGlobalHooks(t, map[string][]map[string]any{
		"post-start": {
			{"exec-host": "echo global >> hooklog.txt"},
			{"exec-host": "echo shared >> hooklog.txt"},
			{"exec": "true", "service": "nonexistent"},
		},
	})
	tmpDir := t.TempDir()
	app, err := NewApp(tmpDir, false)
	require.NoError(t, err)
	app.Name = "globalhookstest"
	app.FailOnHookFail = true
	app.Hooks = map[string][]YAMLTask{"post-start": {
		{"exec-host": "echo project >> hooklog.txt"},
		{"exec-host": "echo shared >> hooklog.txt"},
	}}
	logFile := filepath.Join(tmpDir, "hooklog.txt")

	// "shared" is in both places and runs once, at its project position, and
	// the global exec task for a service this project lacks does not fail the hook.
	require.NoError(t, app.ProcessHooks("post-start"))
	got, err := os.ReadFile(logFile)
	require.NoError(t, err)
	require.Equal(t, "global\nproject\nshared\n", string(got))

	require.NoError(t, os.Remove(logFile))
	app.SkipGlobalHooks = true
	require.NoError(t, app.ProcessHooks("post-start"))
	got, err = os.ReadFile(logFile)
	require.NoError(t, err)
	require.Equal(t, "project\nshared\n", string(got))

	require.NoError(t, os.Remove(logFile))
	SkipHooks = true
	t.Cleanup(func() { SkipHooks = false })
	app.SkipGlobalHooks = false
	require.NoError(t, app.ProcessHooks("post-start"))
	require.NoFileExists(t, logFile)
}

func TestCheckCustomConfigReportsGlobalHooks(t *testing.T) {
	setGlobalHooks(t, map[string][]map[string]any{"post-start": {{"exec": "a"}, {"exec": "b"}}})
	app, err := NewApp(t.TempDir(), false)
	require.NoError(t, err)

	message, hasWarnings := app.CheckCustomConfig(true)
	require.True(t, hasWarnings)
	require.Contains(t, message, "Hooks (global)")
	require.Contains(t, message, "global_config.yaml: hooks (2 post-start)")

	app.SkipGlobalHooks = true
	message, _ = app.CheckCustomConfig(true)
	require.NotContains(t, message, "Hooks (global)")
}
