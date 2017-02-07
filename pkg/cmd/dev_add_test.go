package cmd

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/drud/ddev/pkg/local"
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
	}
	for _, p := range files {
		if !utils.FileExists(p) {
			log.Println(p)
			return false
		}
	}
	return true
}

// TestDevAddBadArgs tests whether the command reacts properly to badly formatted or missing args
func TestDevAddBadArgs(t *testing.T) {
	assert := assert.New(t)
	err := setActiveApp("", "")
	assert.NoError(err)

	// test that you get an error when you run with no args
	args := []string{"add"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "app_name and deploy_name are expected as arguments.")

	// test that you get an error when passing a bad environment name
	args = append(args, "mcsnaggletooth", "smith")

	// testing that you get an error when passing a bad site name
	args[3] = DevTestEnv
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "No legacy site by that name")

	err = setActiveApp(utils.RandomString(10), utils.RandomString(10))
	assert.NoError(err)
	args = []string{"add"}
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "No legacy site by that name")

	err = setActiveApp(utils.RandomString(10), DevTestEnv)
	assert.NoError(err)
	out, err = utils.RunCommand(DrudBin, args)
	assert.Error(err)
	assert.Contains(string(out), "No legacy site by that name")

	err = setActiveApp("", "")
	assert.NoError(err)
}

// TestDevAddScaffoldWP uses the scaffold flag to test that everything needed to run a site locally is created correctly
func TestDevAddScaffoldWP(t *testing.T) {
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"add", DevTestApp, DevTestEnv, "-s"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Successfully added")

	app := local.NewLegacyApp(DevTestApp, DevTestEnv)
	assert.Equal(true, checkRequiredFolders(app))

}

// TestDevAddScaffoldWPImageTag makes sure the --web-image-tag and --db-image-tag flags work
func TestDevAddScaffoldWPImageTag(t *testing.T) {
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"add", DevTestApp, DevTestEnv, "-s", "--web-image-tag=unison,", "--db-image-tag=5.6"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Successfully added")

	app := local.NewLegacyApp(DevTestApp, DevTestEnv)
	assert.Equal(true, checkRequiredFolders(app))

	composeFile, err := ioutil.ReadFile(path.Join(app.AbsPath(), "docker-compose.yaml"))
	assert.NoError(err)

	assert.Contains(string(composeFile), "unison")
	assert.Contains(string(composeFile), "5.6")

}

// TestDevAddScaffoldWPImageChange makes sure the --web-image and --db-image flags work
func TestDevAddScaffoldWPImageChange(t *testing.T) {
	assert := assert.New(t)

	args := []string{"add", DevTestApp, DevTestEnv, "-s",
		"--web-image=drud/testmewebimage,", "--db-image=drud/testmedbimage",
	}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "Successfully added")

	app := local.NewLegacyApp(DevTestApp, DevTestEnv)
	assert.Equal(true, checkRequiredFolders(app))

	composeFile, err := ioutil.ReadFile(path.Join(app.AbsPath(), "docker-compose.yaml"))
	assert.NoError(err)

	assert.Contains(string(composeFile), "drud/testmewebimage")
	assert.Contains(string(composeFile), "drud/testmedbimage")

}

// TestDevAddWP tests a `drud Dev add` on a wp site
func TestDevAddSites(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)
	for _, site := range DevTestSites {

		// test that you get an error when you run with no args
		args := []string{"add", site[0], site[1]}
		out, err := utils.RunCommand(DrudBin, args)
		log.Println("Output from drud dev add:", out)
		assert.NoError(err)
		assert.Contains(string(out), "Successfully added")
		assert.Contains(string(out), "Your application can be reached at")

		app := local.NewLegacyApp(site[0], site[1])

		assert.Equal(true, checkRequiredFolders(app))
		assert.Equal(true, utils.IsRunning(app.ContainerName()+"-web"))
		assert.Equal(true, utils.IsRunning(app.ContainerName()+"-db"))

		o := utils.NewHTTPOptions("http://127.0.0.1")
		o.Timeout = 90
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
