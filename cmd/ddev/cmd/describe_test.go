package cmd

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/stretchr/testify/require"
	"runtime"
	"strings"
	"testing"

	"encoding/json"

	"os"

	"path/filepath"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	log "github.com/sirupsen/logrus"
	asrt "github.com/stretchr/testify/assert"
)

// TestDescribeBadArgs ensures the binary behaves as expected when used with invalid arguments or working directories.
func TestDescribeBadArgs(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it for the duration of this test.
	tmpdir := testcommon.CreateTmpDir("badargs")
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	// Ensure it fails if we run the vanilla describe outside of an application root.
	args := []string{"describe"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Please specify a project name or change directories")

	// Ensure we get a failure if we run a describe on a named application which does not exist.
	args = []string{"describe", util.RandString(16)}
	out, err = exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "could not find requested project")

	// Ensure we get a failure if using too many arguments.
	args = []string{"describe", util.RandString(16), util.RandString(16)}
	out, err = exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(string(out), "Too many arguments provided")

}

// TestCmdDescribe tests that the describe command works properly when using the binary.
func TestCmdDescribe(t *testing.T) {
	assert := asrt.New(t)

	for _, v := range TestSites {
		// First, try to do a describe from another directory.
		tmpdir := testcommon.CreateTmpDir("")
		cleanup := testcommon.Chdir(tmpdir)
		defer testcommon.CleanupDir(tmpdir)

		args := []string{"describe", v.Name}
		out, err := exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "NAME")
		assert.Contains(string(out), "LOCATION")
		assert.Contains(string(out), v.Name)
		assert.Contains(string(out), "running")

		cleanup()

		cleanup = v.Chdir()
		defer cleanup()

		args = []string{"describe"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		assert.Contains(string(out), "NAME")
		assert.Contains(string(out), "LOCATION")
		assert.Contains(string(out), v.Name)
		assert.Contains(string(out), "running")

		// Test describe in current directory with json flag
		args = []string{"describe", "-j"}
		out, err = exec.RunCommand(DdevBin, args)
		assert.NoError(err)
		logItems, err := unmarshalJSONLogs(out)
		require.NoError(t, err, "Unable to unmarshall ===\n%s\n===\n", logItems)

		// The description log should be next last item; there may be a warning
		// or other info before that.
		var raw map[string]interface{}
		rawFound := false
		var item map[string]interface{}
		for _, item = range logItems {
			if item["level"] == "info" {
				if raw, rawFound = item["raw"].(map[string]interface{}); rawFound {
					break
				}
			}
		}
		require.True(t, rawFound, "did not find 'raw' in item in logItems\n===\n%s\n===\n", out)
		assert.EqualValues("running", raw["status"])
		assert.EqualValues(v.Name, raw["name"])
		assert.Equal(ddevapp.RenderHomeRootedDir(v.Dir), raw["shortroot"].(string))
		assert.EqualValues(v.Dir, raw["approot"].(string))

		assert.NotEmpty(item["msg"])
	}
}

// TestCmdDescribeAppFunction performs unit tests on the describeApp function from the working directory.
func TestCmdDescribeAppFunction(t *testing.T) {
	assert := asrt.New(t)
	for i, v := range TestSites {
		cleanup := v.Chdir()

		app, err := ddevapp.GetActiveApp("")
		assert.NoError(err)

		desc, err := app.Describe(false)
		assert.NoError(err)
		assert.EqualValues(ddevapp.SiteRunning, desc["status"])
		assert.EqualValues(app.GetName(), desc["name"])
		assert.EqualValues(ddevapp.RenderHomeRootedDir(v.Dir), desc["shortroot"].(string))
		assert.EqualValues(v.Dir, desc["approot"].(string))
		assert.Equal(app.GetHTTPURL(), desc["httpurl"])
		assert.Equal(app.GetName(), desc["name"])
		assert.Equal("healthy", desc["router_status"], "project #%d %s desc does not have healthy router status", i, app.Name)
		assert.Equal(v.Dir, desc["approot"])

		// Stop the router using docker and then check the describe
		_, err = exec.RunCommand("docker", []string{"stop", "ddev-router"})
		assert.NoError(err)
		desc, err = app.Describe(false)
		assert.NoError(err)
		assert.Equal("exited", desc["router_status"])
		_, err = exec.RunCommand("docker", []string{"start", "ddev-router"})
		assert.NoError(err)

		_, err = dockerutil.ContainerWait(10, map[string]string{"com.docker.compose.service": "ddev-router"})
		assert.NoError(err)

		cleanup()
	}
}

// TestCmdDescribeAppUsingSitename performs unit tests on the describeApp function using the sitename as an argument.
func TestCmdDescribeAppUsingSitename(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it for the duration of this test.
	tmpdir := testcommon.CreateTmpDir("describeAppUsingSitename")
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	for _, v := range TestSites {
		app, err := ddevapp.GetActiveApp(v.Name)
		assert.NoError(err)
		desc, err := app.Describe(false)
		assert.NoError(err)
		assert.EqualValues(desc["status"], ddevapp.SiteRunning)
		assert.EqualValues(app.GetName(), desc["name"])
		assert.EqualValues(ddevapp.RenderHomeRootedDir(v.Dir), desc["shortroot"].(string))
		assert.EqualValues(v.Dir, desc["approot"].(string))

		out, _ := json.Marshal(desc)
		assert.NoError(err)
		assert.Contains(string(out), "running")
		assert.Contains(string(out), ddevapp.RenderHomeRootedDir(v.Dir))
	}
}

// TestCmdDescribeAppWithInvalidParams performs unit tests on the describeApp function using a variety of invalid parameters.
func TestCmdDescribeAppWithInvalidParams(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and switch to it for the duration of this test.
	tmpdir := testcommon.CreateTmpDir("TestCmdDescribeAppWithInvalidParams")
	defer testcommon.CleanupDir(tmpdir)
	defer testcommon.Chdir(tmpdir)()

	// Ensure describeApp fails from an invalid working directory.
	_, err := ddevapp.GetActiveApp("")
	assert.Error(err)

	// Ensure describeApp fails with invalid site-names.
	_, err = ddevapp.GetActiveApp(util.RandString(16))
	assert.Error(err)

	// Change to a site's working directory and ensure a failure still occurs with a invalid site name.
	cleanup := TestSites[0].Chdir()
	_, err = ddevapp.GetActiveApp(util.RandString(16))
	assert.Error(err)
	cleanup()
}

// unmarshalJSONLogs takes a string buffer and splits it into lines,
// discards empty lines, and unmarshalls into an array of logs
func unmarshalJSONLogs(in string) ([]log.Fields, error) {
	logData := make([]log.Fields, 0)
	logStrings := strings.Split(in, "\n")

	for _, logLine := range logStrings {
		if logLine != "" {
			data := make(log.Fields, 4)
			err := json.Unmarshal([]byte(logLine), &data)
			if err != nil {
				return []log.Fields{}, err
			}
			logData = append(logData, data)
		}
	}
	return logData, nil
}

// TestCmdDescribeMissingProjectDirectory ensures the `ddev describe` command returns the expected help text when
// a project's directory no longer exists.
func TestCmdDescribeMissingProjectDirectory(t *testing.T) {
	var err error
	var out string

	if runtime.GOOS == "windows" {
		t.Skip("Skipping because unreliable on Windows")
	}

	assert := asrt.New(t)

	projDir, _ := os.Getwd()

	projectName := util.RandString(6)

	tmpDir := testcommon.CreateTmpDir(t.Name())
	defer testcommon.CleanupDir(tmpDir)
	defer testcommon.Chdir(tmpDir)()

	_, err = exec.RunCommand(DdevBin, []string{"config", "--project-type", "php", "--project-name", projectName})
	assert.NoError(err)

	_, err = exec.RunCommand(DdevBin, []string{"start", "-y"})
	assert.NoError(err)

	t.Cleanup(func() {
		_, err = exec.RunCommand(DdevBin, []string{"delete", "-Oy", projectName})
		assert.NoError(err)
	})

	_, err = exec.RunCommand(DdevBin, []string{"stop"})
	assert.NoError(err)

	err = os.Chdir(projDir)
	assert.NoError(err)
	copyDir := filepath.Join(testcommon.CreateTmpDir(t.Name()), util.RandString(4))
	err = os.Rename(tmpDir, copyDir)
	assert.NoError(err)

	out, err = exec.RunCommand(DdevBin, []string{"describe", projectName})
	assert.Error(err, "Expected an error when describing project with no project directory")
	assert.Contains(out, "ddev can no longer find your project files")
	err = os.Rename(copyDir, tmpDir)
	assert.NoError(err)
}
