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
// vars but does NOT set them in the process environment. This is the fix for ddev/ddev#472.
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
