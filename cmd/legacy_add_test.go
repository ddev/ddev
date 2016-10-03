package cmd

import (
	"fmt"
	"net"
	"path"
	"testing"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/bootstrap/cli/utils"
	drudutils "github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

var (
	drudBin       = "drud"
	legacyTestApp = "drudio"
	legacyTestEnv = "production"
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
		if !drudutils.FileExists(p) {
			return false
		}
	}
	return true
}

// // @todo: move me to shared package
func IsTCPPortAvailable(port int) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// TestLegacyAddBadArgs tests whether the command reacts properly to badly formatted or missing args
func TestLegacyAddBadArgs(t *testing.T) {
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"legacy", "add"}
	out, err := drudutils.RunCommand(drudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "app_name and deploy_name are expected as arguments.")

	// test that you get an error when passing a bad environment name
	args = append(args, "mcsnaggletooth", "smith")
	out, err = drudutils.RunCommand(drudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Bad environment name.")

	// testing that you get an error when passing a bad site name
	args[3] = legacyTestEnv
	out, err = drudutils.RunCommand(drudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "No legacy site by that name")
}

// TestLegacyAddScaffoldWP uses the scaffold flag to test that everything needed to run a site locally is created correctly
func TestLegacyAddScaffoldWP(t *testing.T) {
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"legacy", "add", legacyTestApp, legacyTestEnv, "-s", "-t", "wp"}
	out, err := drudutils.RunCommand(drudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Successfully added")

	app := local.LegacyApp{
		Name:        legacyTestApp,
		Environment: legacyTestEnv,
	}

	assert.Equal(true, checkRequiredFolders(app))

}

// TestLegacyAddWP tests a `drud legacy add` on a wp site
func TestLegacyAddWP(t *testing.T) {
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"legacy", "add", legacyTestApp, legacyTestEnv, "-t", "wp"}
	out, err := drudutils.RunCommand(drudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Successfully added")
	assert.Contains(string(out), "WordPress site")

	app := local.LegacyApp{
		Name:        legacyTestApp,
		Environment: legacyTestEnv,
	}

	assert.Equal(true, checkRequiredFolders(app))
	assert.Equal(true, utils.IsRunning(app.ContainerName()+"-web"))
	assert.Equal(true, utils.IsRunning(app.ContainerName()+"-db"))

	webPort, err := local.GetPodPort(app.ContainerName() + "-web")
	assert.NoError(err)
	dbPort, err := local.GetPodPort(app.ContainerName() + "-db")
	assert.NoError(err)

	assert.Equal(true, IsTCPPortAvailable(int(webPort)))
	assert.Equal(true, IsTCPPortAvailable(int(dbPort)))
	err = drudutils.EnsureHTTPStatus(fmt.Sprintf("http://localhost:%d", webPort), "", "", 120, 200)
	assert.NoError(err)

}
