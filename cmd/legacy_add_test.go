package cmd

import (
	"fmt"
	"path"
	"testing"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

func checkRequiredFolders(app local.App) bool {
	basePath := app.AbsPath()
	files := []string{
		basePath,
		path.Join(basePath, "src"),
		path.Join(basePath, "data"),
		path.Join(basePath, "data", app.GetName()+".sql"),
		path.Join(basePath, "files"),
		path.Join(basePath, "docker-compose.yaml"),
	}
	for _, p := range files {
		if !utils.FileExists(p) {
			return false
		}
	}
	return true
}

// TestLegacyAddBadArgs tests whether the command reacts properly to badly formatted or missing args
func TestLegacyAddBadArgs(t *testing.T) {
	assert := assert.New(t)
	err := setActiveApp("", "")
	assert.NoError(err)

	// test that you get an error when you run with no args
	args := []string{"legacy", "add"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "app_name and deploy_name are expected as arguments.")

	// test that you get an error when passing a bad environment name
	args = append(args, "mcsnaggletooth", "smith")
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Bad environment name.")

	// testing that you get an error when passing a bad site name
	args[3] = LegacyTestEnv
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "No legacy site by that name")

	err = setActiveApp(utils.RandomString(10), utils.RandomString(10))
	assert.NoError(err)
	args = []string{"legacy", "add"}
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Bad environment name")

	err = setActiveApp(utils.RandomString(10), LegacyTestEnv)
	assert.NoError(err)
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "No legacy site by that name")

	err = setActiveApp("", "")
	assert.NoError(err)
}

// TestLegacyAddScaffoldWP uses the scaffold flag to test that everything needed to run a site locally is created correctly
func TestLegacyAddScaffoldWP(t *testing.T) {
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"legacy", "add", LegacyTestApp, LegacyTestEnv, "-s", "-t", "wp"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Successfully added")

	app := local.LegacyApp{
		Name:        LegacyTestApp,
		Environment: LegacyTestEnv,
	}

	assert.Equal(true, checkRequiredFolders(app))

}

// TestLegacyAddWP tests a `drud legacy add` on a wp site
func TestLegacyAddWP(t *testing.T) {
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"legacy", "add", LegacyTestApp, LegacyTestEnv, "-t", "wp"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Successfully added")
	assert.Contains(string(out), "Your application can be reached at")

	app := local.LegacyApp{
		Name:        LegacyTestApp,
		Environment: LegacyTestEnv,
	}

	assert.Equal(true, checkRequiredFolders(app))
	assert.Equal(true, utils.IsRunning(app.ContainerName()+"-web"))
	assert.Equal(true, utils.IsRunning(app.ContainerName()+"-db"))

	webPort, err := local.GetPodPort(app.ContainerName() + "-web")
	assert.NoError(err)
	dbPort, err := local.GetPodPort(app.ContainerName() + "-db")
	assert.NoError(err)

	assert.Equal(true, utils.IsTCPPortAvailable(int(webPort)))
	assert.Equal(true, utils.IsTCPPortAvailable(int(dbPort)))
	err = utils.EnsureHTTPStatus(fmt.Sprintf("http://localhost:%d", webPort), "", "", 120, 200)
	assert.NoError(err)

}
