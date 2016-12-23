package cmd

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

var skipComposeTests bool

func checkRequiredFolders(app local.App) bool {
	basePath := app.AbsPath()
	files := []string{
		basePath,
		path.Join(basePath, "src"),
		path.Join(basePath, "data"),
		path.Join(basePath, "data", "data.sql"),
		path.Join(basePath, "files"),
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
	args := []string{"dev", "add"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "app_name and deploy_name are expected as arguments.")

	// test that you get an error when passing a bad environment name
	args = append(args, "mcsnaggletooth", "smith")

	// testing that you get an error when passing a bad site name
	args[3] = LegacyTestEnv
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "No legacy site by that name")

	err = setActiveApp(utils.RandomString(10), utils.RandomString(10))
	assert.NoError(err)
	args = []string{"dev", "add"}
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "No legacy site by that name")

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
	args := []string{"dev", "add", LegacyTestApp, LegacyTestEnv, "-s"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Successfully added")

	app := local.NewLegacyApp(LegacyTestApp, LegacyTestEnv)
	assert.Equal(true, checkRequiredFolders(app))

}

// TestLegacyAddScaffoldWPImageTag makes sure the --web-image-tag and --db-image-tag flags work
func TestLegacyAddScaffoldWPImageTag(t *testing.T) {
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"dev", "add", LegacyTestApp, LegacyTestEnv, "-s", "--web-image-tag=unison,", "--db-image-tag=5.6"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Successfully added")

	app := local.NewLegacyApp(LegacyTestApp, LegacyTestEnv)
	assert.Equal(true, checkRequiredFolders(app))

	composeFile, err := ioutil.ReadFile(path.Join(app.AbsPath(), "docker-compose.yaml"))
	assert.NoError(err)

	assert.Contains(string(composeFile), "unison")
	assert.Contains(string(composeFile), "5.6")

}

// TestLegacyAddScaffoldWPImageChange makes sure the --web-image and --db-image flags work
func TestLegacyAddScaffoldWPImageChange(t *testing.T) {
	assert := assert.New(t)

	args := []string{"dev", "add", LegacyTestApp, LegacyTestEnv, "-s",
		"--web-image=drud/testmewebimage,", "--db-image=drud/testmedbimage",
	}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Successfully added")

	app := local.NewLegacyApp(LegacyTestApp, LegacyTestEnv)
	assert.Equal(true, checkRequiredFolders(app))

	composeFile, err := ioutil.ReadFile(path.Join(app.AbsPath(), "docker-compose.yaml"))
	assert.NoError(err)

	assert.Contains(string(composeFile), "drud/testmewebimage")
	assert.Contains(string(composeFile), "drud/testmedbimage")

}

// TestLegacyAddWP tests a `drud legacy add` on a wp site
func TestLegacyAddSites(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)
	for _, site := range LegacyTestSites {

		// test that you get an error when you run with no args
		args := []string{"dev", "add", site[0], site[1]}
		out, err := utils.RunCommand(DrudBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "Successfully added")
		assert.Contains(string(out), "Your application can be reached at")

		app := local.NewLegacyApp(site[0], site[1])

		assert.Equal(true, checkRequiredFolders(app))
		assert.Equal(true, utils.IsRunning(app.ContainerName()+"-web"))
		assert.Equal(true, utils.IsRunning(app.ContainerName()+"-db"))

		webPort, err := local.GetPodPort(app.ContainerName() + "-web")
		assert.NoError(err)
		dbPort, err := local.GetPodPort(app.ContainerName() + "-db")
		assert.NoError(err)

		assert.Equal(true, utils.IsTCPPortAvailable(int(webPort)))
		assert.Equal(true, utils.IsTCPPortAvailable(int(dbPort)))
		o := utils.NewHTTPOptions("http://127.0.0.1")
		o.Timeout = 120
		o.Headers["Host"] = app.HostName()
		err = utils.EnsureHTTPStatus(o)
		assert.NoError(err)
	}
}

func TestSubTag(t *testing.T) {
	tests := [][]string{
		[]string{"drud/testimage", "drud/testimage:unison"},
		[]string{"drud/testimage:test", "drud/testimage:unison"},
		[]string{"http://docker.io/drud/testimage:test", "http://docker.io/drud/testimage:unison"},
		[]string{"http://docker.io/drud/testimage", "http://docker.io/drud/testimage:unison"},
		[]string{"http://docker.io/drud/testimage:unison", "http://docker.io/drud/testimage:unison"},
	}
	for _, l := range tests {
		img := local.SubTag(l[0], "unison")
		assert.Equal(t, l[1], img)
	}
}

func init() {
	if os.Getenv("SKIP_COMPOSE_TESTS") == "true" {
		skipComposeTests = true
	}
}
