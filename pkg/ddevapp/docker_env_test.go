package ddevapp_test

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestDockerEnvDoesNotMutateProcessEnv asserts that DockerEnv() returns project-specific
// vars but does NOT set them in the process environment (fixes ddev/ddev#472).
func TestDockerEnvDoesNotMutateProcessEnv(t *testing.T) {
	testcommon.ClearDockerEnv()

	before := map[string]string{
		"DDEV_SITENAME":        os.Getenv("DDEV_SITENAME"),
		"COMPOSE_PROJECT_NAME": os.Getenv("COMPOSE_PROJECT_NAME"),
	}

	projectName := "TestDockerEnv472"
	_ = globalconfig.RemoveProjectInfo(projectName)

	testDir := testcommon.CreateTmpDir(projectName)
	defer testcommon.CleanupDir(testDir)
	defer func() { _ = globalconfig.RemoveProjectInfo(projectName) }()

	app, err := ddevapp.NewApp(testDir, true)
	require.NoError(t, err)
	app.Name = projectName
	app.Type = nodeps.AppTypePHP
	err = app.WriteConfig()
	require.NoError(t, err)

	envMap := app.DockerEnv()
	require.NotEmpty(t, envMap["DDEV_SITENAME"])
	require.NotEmpty(t, envMap["COMPOSE_PROJECT_NAME"])

	for key, expected := range before {
		actual := os.Getenv(key)
		require.Equal(t, expected, actual, "DockerEnv() must not mutate process env for %s (fixes #472)", key)
	}
}

// TestComposeSubprocessReceivesOnlyOverrides asserts that the compose subprocess receives
// exactly the override values from BuildProjectEnv and does not inherit conflicting vars
// from the parent process. This guards against env leakage (ddev/ddev#472).
func TestComposeSubprocessReceivesOnlyOverrides(t *testing.T) {
	testcommon.ClearDockerEnv()

	// Intentionally pollute parent env with values that would leak if overrides failed.
	err := os.Setenv("DDEV_SITENAME", "leaked_from_parent")
	require.NoError(t, err)
	defer func() { _ = os.Unsetenv("DDEV_SITENAME") }()

	projectName := "correct_value"
	_ = globalconfig.RemoveProjectInfo(projectName)

	testDir := testcommon.CreateTmpDir("TestComposeSubprocessOverrides")
	defer testcommon.CleanupDir(testDir)
	defer func() { _ = globalconfig.RemoveProjectInfo(projectName) }()

	app, err := ddevapp.NewApp(testDir, true)
	require.NoError(t, err)
	app.Name = projectName
	app.Type = nodeps.AppTypePHP
	err = app.WriteConfig()
	require.NoError(t, err)

	// WriteDockerComposeYAML runs compose config via ComposeCmdWithProjectEnv; the compose
	// subprocess must use override values, not the parent's leaked value.
	err = app.WriteDockerComposeYAML()
	require.NoError(t, err)

	rendered, err := os.ReadFile(app.DockerComposeFullRenderedYAMLPath())
	require.NoError(t, err)
	renderedStr := string(rendered)

	// Rendered config must contain the override value (from BuildProjectEnv) in container names.
	require.Contains(t, renderedStr, "ddev-"+projectName+"-", "compose output must use override DDEV_SITENAME")
	// Must NOT contain the parent's leaked value.
	require.NotContains(t, renderedStr, "leaked_from_parent", "compose subprocess must not inherit parent env")
}
