package dockerutil_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var DdevBin = "ddev"

func init() {
	// Make sets DDEV_BINARY_FULLPATH when building the executable
	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}
	if os.Getenv("DDEV_TEST_NO_BIND_MOUNTS") == "true" {
		globalconfig.DdevGlobalConfig.NoBindMounts = true
	}

	globalconfig.EnsureGlobalConfig()
}

// TestDockerBuildxDownload verifies that we can download a particular docker-buildx version
func TestDockerBuildxDownload(t *testing.T) {
	_, err := dockerutil.DownloadDockerBuildxIfNeeded()
	require.NoError(t, err)

	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)

	t.Cleanup(func() {
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
	})

	// Remove previous binary
	previousDockerBuildx, _ := globalconfig.GetDockerBuildxDestination()
	_ = os.RemoveAll(previousDockerBuildx)

	downloaded, err := dockerutil.DownloadDockerBuildxIfNeeded(true)
	require.NoError(t, err)
	require.True(t, downloaded)
	v, err := dockerutil.GetDockerBuildxVersion()
	require.NoError(t, err)
	require.Equal(t, globalconfig.GetRequiredDockerBuildxVersion(), v)

	// Make sure it doesn't download a second time
	downloaded, err = dockerutil.DownloadDockerBuildxIfNeeded(true)
	require.NoError(t, err)
	require.False(t, downloaded)

	for _, v := range []string{"0.32.0"} {
		globalconfig.DdevGlobalConfig.RequiredDockerBuildxVersion = v
		downloaded, err = dockerutil.DownloadDockerBuildxIfNeeded(true)
		require.NoError(t, err)
		require.True(t, downloaded)
		activeVersion, err := dockerutil.GetDockerBuildxVersion()
		require.NoError(t, err)
		require.Equal(t, globalconfig.GetRequiredDockerBuildxVersion(), activeVersion)
	}
}

// TestComposeConfig tests ComposeConfig which loads and merges compose files.
func TestComposeConfig(t *testing.T) {
	assert := asrt.New(t)

	// Create a dummy app to set up DockerEnv
	testDir := testcommon.CreateTmpDir(t.Name())
	app, err := ddevapp.NewApp(testDir, true)
	assert.NoError(err)

	t.Cleanup(func() {
		_ = os.RemoveAll(testDir)
	})

	app.Name = "test"

	// Set up environment variables via DockerEnv
	_ = app.DockerEnv()

	composeFiles := []string{filepath.Join("testdata", "docker-compose.yml")}

	// Test: should return project with services
	project, err := dockerutil.ComposeConfig(dockerutil.ComposeConfigOpts{
		ComposeFiles: composeFiles,
		ProjectName:  "test",
	})
	assert.NoError(err)
	assert.NotNil(project)
	_, hasWeb := project.Services["web"]
	assert.True(hasWeb)
	_, hasDB := project.Services["db"]
	assert.True(hasDB)

	// Test with additional override file
	composeFiles = append(composeFiles, filepath.Join("testdata", "docker-compose.override.yml"))
	project, err = dockerutil.ComposeConfig(dockerutil.ComposeConfigOpts{
		ComposeFiles: composeFiles,
		ProjectName:  "test",
	})
	assert.NoError(err)
	assert.NotNil(project)
	_, hasWeb = project.Services["web"]
	assert.True(hasWeb)
	_, hasDB = project.Services["db"]
	assert.True(hasDB)
	_, hasFoo := project.Services["foo"]
	assert.True(hasFoo)

	// Test with invalid file
	_, err = dockerutil.ComposeConfig(dockerutil.ComposeConfigOpts{
		ComposeFiles: []string{"invalid.yml"},
		ProjectName:  "test",
	})
	assert.Error(err)
}

// TestComposeExec tests execution of docker-compose exec commands with streams
func TestComposeExec(t *testing.T) {
	assert := asrt.New(t)

	projectName := strings.ToLower(t.Name())
	// Container name and labels come from the compose file
	containerName := projectName

	container, _ := dockerutil.FindContainerByName(containerName)
	if container != nil {
		_ = dockerutil.RemoveContainer(container.ID)
	}

	// Use the current actual web container for this, so replace in base docker-compose file
	composeBase := filepath.Join("testdata", "TestComposeExec", "test-compose-with-streams.yaml")
	tmpDir := testcommon.CreateTmpDir(t.Name())
	realComposeFile := filepath.Join(tmpDir, "replaced-compose-with-streams.yaml")

	err := fileutil.ReplaceStringInFile("TEST-COMPOSE-EXEC-IMAGE", versionconstants.WebImg+":"+versionconstants.WebTag, composeBase, realComposeFile)
	assert.NoError(err)

	composeFiles := []string{realComposeFile}

	t.Cleanup(func() {
		_ = dockerutil.ComposeDown(dockerutil.ComposeDownOpts{
			ComposeFiles: composeFiles,
			ProjectName:  projectName,
		})
	})

	err = dockerutil.ComposeUp(dockerutil.ComposeUpOpts{
		ComposeFiles: composeFiles,
		ProjectName:  projectName,
	})
	require.NoError(t, err)

	_, err = dockerutil.ContainerWait(60, map[string]string{
		"com.ddev.site-name":        containerName,
		"com.docker.compose.oneoff": "False",
	})
	if err != nil {
		logout, _ := exec.RunCommand("docker", []string{"logs", containerName})
		inspectOut, _ := exec.RunCommandPipe("sh", []string{"-c", fmt.Sprintf("docker inspect %s|jq -r '.[0].State.Health.Log'", containerName)})
		t.Fatalf("FAIL: TestComposeExec failed to ContainerWait for container: %v, logs\n========= container logs ======\n%s\n======= end logs =======\n==== health log =====\ninspectOut\n%s\n========", err, logout, inspectOut)
	}

	// Point stdout to os.Stdout and do simple ps -ef in web container
	stdout := util.CaptureStdOut()
	_, _, err = dockerutil.ComposeExec(dockerutil.ComposeExecOpts{
		ComposeFiles: composeFiles,
		ProjectName:  projectName,
		Service:      "web",
		Command:      []string{"ps", "-ef"},
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	})
	assert.NoError(err)
	output := stdout()
	assert.Contains(output, "supervisord")

	// Reverse stdout and stderr: error output should appear on our captured stdout
	stdout = util.CaptureStdOut()
	_, _, err = dockerutil.ComposeExec(dockerutil.ComposeExecOpts{
		ComposeFiles: composeFiles,
		ProjectName:  projectName,
		Service:      "web",
		Command:      []string{"ls", "-d", "xx", "/var/run/apache2"},
		Stdout:       os.Stderr, // swapped
		Stderr:       os.Stdout, // swapped
	})
	assert.Error(err)
	output = stdout()
	assert.Contains(output, "ls: cannot access 'xx': No such file or directory")

	// Normal stdout/stderr: success output should appear on our captured stdout
	stdout = util.CaptureStdOut()
	_, _, err = dockerutil.ComposeExec(dockerutil.ComposeExecOpts{
		ComposeFiles: composeFiles,
		ProjectName:  projectName,
		Service:      "web",
		Command:      []string{"ls", "-d", "xx", "/var/run/apache2"},
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	})
	assert.Error(err)
	output = stdout()
	assert.Contains(output, "/var/run/apache2", output)
}
