package dockerutil_test

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testContainerName = "TestDockerUtils"

func init() {
	globalconfig.EnsureGlobalConfig()
	dockerutil.EnsureDdevNetwork()
}

func TestMain(m *testing.M) { os.Exit(testMain(m)) }

func testMain(m *testing.M) int {
	testcommon.ClearDockerEnv()

	_ = os.Setenv("DDEV_NONINTERACTIVE", "true")
	_ = os.Setenv("MUTAGEN_DATA_DIRECTORY", globalconfig.GetMutagenDataDirectory())

	labels := map[string]string{
		"com.ddev.site-name":        testContainerName,
		"com.docker.compose.oneoff": "False",
	}

	// Prep Docker container for Docker util tests
	imageExists, err := dockerutil.ImageExistsLocally(ddevImages.GetWebImage())
	if err != nil {
		output.UserErr.Errorf("Failed to check for local image %s: %v", ddevImages.GetWebImage(), err)
		return 6
	}
	if !imageExists {
		err := dockerutil.Pull(ddevImages.GetWebImage())
		if err != nil {
			output.UserErr.Errorf("Failed to pull test image: %v", err)
			return 7
		}
	}

	foundContainer, _ := dockerutil.FindContainerByLabels(labels)

	if foundContainer != nil {
		err = dockerutil.RemoveContainer(foundContainer.ID)
		if err != nil {
			output.UserErr.Errorf("-- FAIL: dockerutils_test TestMain failed to remove container %s: %v", foundContainer.ID, err)
			return 5
		}
	}

	containerID, err := startTestContainer()
	if err != nil {
		output.UserErr.Errorf("-- FAIL: dockerutils_test failed to start test container: %v", err)
		return 3
	}
	defer func() {
		foundContainer, _ := dockerutil.FindContainerByLabels(labels)
		if foundContainer != nil {
			err = dockerutil.RemoveContainer(foundContainer.ID)
			if err != nil {
				output.UserErr.Warningf("-- FAIL: dockerutils_test failed to remove test container '%v': %v", foundContainer, err)
			}
		}
	}()

	output.UserOut.Printf("ContainerWait at %v", time.Now())
	out, err := dockerutil.ContainerWait(60, map[string]string{
		"com.ddev.site-name":        testContainerName,
		"com.docker.compose.oneoff": "False",
	})
	output.UserOut.Printf("ContainerWait returned at %v out='%s' err='%v'", time.Now(), out, err)

	if err != nil {
		logout, _ := exec.RunHostCommand("docker", "logs", containerID)
		inspectOut, _ := exec.RunHostCommand("sh", "-c", fmt.Sprintf("docker inspect %s|jq -r '.[0].State.Health.Log'", containerID))
		output.UserErr.Printf("FAIL: dockerutils_test testMain failed to ContainerWait for container: %v, logs\n========= container logs ======\n%s\n======= end logs =======\n==== health log =====\ninspectOut\n%s\n========", err, logout, inspectOut)
		return 4
	}
	exitStatus := m.Run()

	return exitStatus
}

// Start container for tests; returns containerID, error
func startTestContainer() (string, error) {
	portBinding := map[network.Port][]network.PortBinding{
		network.MustParsePort("80/tcp"): {
			{HostPort: "8889"},
		},
		network.MustParsePort("8025/tcp"): {
			{HostPort: "8890"},
		}}
	containerID, _, err := dockerutil.RunSimpleContainer(ddevImages.GetWebImage(), testContainerName, nil, nil, []string{
		"HOTDOG=superior-to-corndog",
		"POTATO=future-fry",
		"DDEV_WEBSERVER_TYPE=nginx-fpm",
	}, nil, "33", false, true, map[string]string{
		"com.docker.compose.service": "web",
		"com.ddev.site-name":         testContainerName,
		"com.docker.compose.oneoff":  "False",
	}, portBinding, nil)
	if err != nil {
		return "", err
	}

	return containerID, nil
}

// TestGetContainerUser to make sure that the user provisioned in the container has
// the proper uid/gid/username characteristics.
func TestGetContainerUser(t *testing.T) {
	origDir, _ := os.Getwd()

	// Create temporary directory for test project
	testDir := testcommon.CreateTmpDir(t.Name())
	projName := t.Name()

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		app, err := ddevapp.GetActiveApp(projName)
		if err == nil {
			_ = app.Stop(true, false)
		}
		_ = os.RemoveAll(testDir)
		testcommon.ClearDockerEnv()
	})

	// Clean up any existing name conflicts
	app, err := ddevapp.GetActiveApp(projName)
	if err == nil {
		err = app.Stop(true, false)
		require.NoError(t, err)
	}

	// Create new app
	app, err = ddevapp.NewApp(testDir, false)
	require.NoError(t, err)
	app.Type = nodeps.AppTypePHP
	app.Name = projName
	err = app.WriteConfig()
	require.NoError(t, err)

	defer util.TimeTrackC(fmt.Sprintf("%s %s", projName, t.Name()))()

	err = app.Restart()
	require.NoError(t, err)

	uid, gid, username := dockerutil.GetContainerUser()

	for _, service := range []string{"web", "db"} {
		out, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: service,
			Cmd:     "id -un",
		})
		require.NoError(t, err)
		require.Equal(t, username, strings.Trim(out, "\r\n"))

		out, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: service,
			Cmd:     "id -u",
		})
		require.NoError(t, err)
		require.Equal(t, uid, strings.Trim(out, "\r\n"))

		out, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: service,
			Cmd:     "id -g",
		})
		require.NoError(t, err)
		require.Equal(t, gid, strings.Trim(out, "\r\n"))
	}
}

// TestGetContainerHealth tests the function for processing container readiness.
func TestGetContainerHealth(t *testing.T) {
	assert := asrt.New(t)
	ctx, apiClient, err := dockerutil.GetDockerClient()
	if err != nil {
		t.Fatalf("Could not get docker client: %v", err)
	}

	labels := map[string]string{
		"com.ddev.site-name":        testContainerName,
		"com.docker.compose.oneoff": "False",
	}

	t.Cleanup(func() {
		c, err := dockerutil.FindContainerByLabels(labels)
		if err == nil || c != nil {
			err = dockerutil.RemoveContainer(c.ID)
			assert.NoError(err)
		}

		// Make sure test container exists again
		_, err = startTestContainer()
		assert.NoError(err)
		healthDetail, err := dockerutil.ContainerWait(30, labels)
		assert.NoError(err, "healthDetail='%s'", healthDetail)

		c, err = dockerutil.FindContainerByLabels(labels)
		assert.NoError(err)
		assert.NotNil(c)

		status, healthDetail := dockerutil.GetContainerHealth(c)
		assert.Contains(healthDetail, "/var/www/html:OK mailpit:OK phpstatus:OK")
		assert.Equal("healthy", status)
	})

	c, err := dockerutil.FindContainerByLabels(labels)
	require.NoError(t, err)
	require.NotNil(t, c)

	status, log := dockerutil.GetContainerHealth(c)
	assert.Equal("healthy", status, "container should be healthy; log=%v", log)

	// Now stop the container and make sure it's exited
	timeout := 10
	_, err = apiClient.ContainerStop(ctx, c.ID, client.ContainerStopOptions{Timeout: &timeout})
	assert.NoError(err)

	status, log = dockerutil.GetContainerHealth(c)
	assert.Equal("exited", status, "container should be exited; log=%v", log)
	assert.NoError(err)
}

// TestContainerWait tests the error cases for the container check wait loop.
func TestContainerWait(t *testing.T) {
	assert := asrt.New(t)

	labels := map[string]string{
		"com.ddev.site-name":        testContainerName,
		"com.docker.compose.oneoff": "False",
	}

	// We should have `testContainerName' already running, it was started by
	// startTestContainer(). And we don't delete it.

	// Try a zero-wait, should show timed-out
	_, err := dockerutil.ContainerWait(0, labels)
	require.Error(t, err)
	require.Contains(t, err.Error(), "health check timed out")

	// Try 30-second wait for "healthy", should show OK
	healthDetail, err := dockerutil.ContainerWait(30, labels)
	require.NoError(t, err)

	require.Contains(t, healthDetail, "phpstatus:OK")

	// Try a nonexistent container, should get error
	labels = map[string]string{"com.ddev.site-name": "nothing-there"}
	// Make sure none already exist
	_ = dockerutil.RemoveContainersByLabels(labels)

	_, err = dockerutil.ContainerWait(1, labels)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to query container")

	// If we run a quick container, and it immediately exits, ContainerWait should find it is not there
	// and note that it exited.
	labels = map[string]string{"test": "quickexit"}
	// Make sure none already exist
	_ = dockerutil.RemoveContainersByLabels(labels)
	c1, _, err := dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, t.Name()+"-quickexit-"+util.RandString(5), []string{"ls"}, nil, nil, nil, "0", false, true, labels, nil, nil)
	t.Cleanup(func() {
		_ = dockerutil.RemoveContainer(c1)
		assert.NoError(err, "failed to remove container %v (%s)", labels, c1)
	})
	require.NoError(t, err)
	_, err = dockerutil.ContainerWait(5, labels)
	require.Error(t, err)
	require.Contains(t, err.Error(), "container exited")

	// If we run a container that does not have a healthcheck
	// it should be found as good immediately
	labels = map[string]string{"test": "nohealthcheck"}
	// Make sure none already exist
	_ = dockerutil.RemoveContainersByLabels(labels)

	c2, _, err := dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, t.Name()+"-nohealthcheck-busybox-sleep-60-"+util.RandString(5), []string{"sleep", "60"}, nil, nil, nil, "0", false, true, labels, nil, nil)
	t.Cleanup(func() {
		err = dockerutil.RemoveContainer(c2)
		assert.NoError(err, "failed to remove container (sleep 60) %v (%s)", labels, c2)
	})
	require.NoError(t, err)
	_, err = dockerutil.ContainerWait(5, labels)
	require.NoError(t, err)

	ddevWebserver := ddevImages.GetWebImage()
	// If we run a container that *does* have a healthcheck but it's unhealthy
	// then ContainerWait shouldn't return until specified wait, and should fail
	// Use ddev-webserver for this; it won't have good health on normal run
	labels = map[string]string{"test": "hashealthcheckbutunhealthy"}
	// Make sure none already exist
	_ = dockerutil.RemoveContainersByLabels(labels)
	c3, _, err := dockerutil.RunSimpleContainer(ddevWebserver, t.Name()+"-hashealthcheckbutunhealthy-"+util.RandString(5), []string{"sleep", "5"}, nil, []string{"DDEV_WEBSERVER_TYPE=nginx-fpm"}, nil, "0", false, true, labels, nil, nil)
	t.Cleanup(func() {
		err = dockerutil.RemoveContainer(c3)
		assert.NoError(err, "failed to remove container %v (%s)", labels, c3)
	})
	require.NoError(t, err)
	_, err = dockerutil.ContainerWait(3, labels)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out without becoming healthy")

	// If we run a container that *does* have a healthcheck but it's not healthy for a while
	// then ContainerWait should detect failure early, but should succeed later
	labels = map[string]string{"test": "hashealthcheckbutbad"}
	// Make sure none already exist
	_ = dockerutil.RemoveContainersByLabels(labels)
	c4, _, err := dockerutil.RunSimpleContainer(ddevWebserver, t.Name()+"-hashealthcheckbutbad2-"+util.RandString(5), []string{"bash", "-c", "sleep 5 && /start.sh"}, nil, []string{"DDEV_WEBSERVER_TYPE=nginx-fpm"}, nil, "0", false, true, labels, nil, nil)
	t.Cleanup(func() {
		err = dockerutil.RemoveContainer(c4)
		assert.NoError(err, "failed to remove container %v (%s)", labels, c4)
	})
	require.NoError(t, err)
	_, err = dockerutil.ContainerWait(3, labels)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out without becoming healthy")
	// Try it again, wait 60s for health; on macOS it usually takes about 2s for ddev-webserver to become healthy
	out, err := dockerutil.ContainerWait(60, labels)
	require.NoError(t, err, "output=%s", out)
}

// TestGetAppContainers looks for container with sitename dockerutils-test
func TestGetAppContainers(t *testing.T) {
	assert := asrt.New(t)
	containers, err := dockerutil.GetAppContainers(testContainerName)
	assert.NoError(err)
	assert.Contains(containers[0].Image, versionconstants.WebImg)
}

// TestFindContainerByName does a simple test of FindContainerByName()
func TestFindContainerByName(t *testing.T) {
	assert := asrt.New(t)

	pwd, _ := os.Getwd()
	pwd, _ = filepath.Abs(pwd)
	testdata := filepath.Join(pwd, "testdata")
	assert.DirExists(testdata)
	containerName := t.Name() + fileutil.RandomFilenameBase()

	// Make sure we don't already have one running
	c, err := dockerutil.FindContainerByName(containerName)
	assert.NoError(err)
	if err == nil && c != nil {
		err = dockerutil.RemoveContainer(c.ID)
		assert.NoError(err)
	}

	// Run a container, don't remove it.
	cID, _, err := dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, containerName, []string{"sleep", "2"}, nil, nil, nil, "25", false, false, nil, nil, nil)
	require.NoError(t, err)

	defer func() {
		_ = dockerutil.RemoveContainer(cID)
	}()

	// Now find the container by name
	c, err = dockerutil.FindContainerByName(containerName)
	assert.NoError(err)
	require.NotNil(t, c)
	// Remove it
	err = dockerutil.RemoveContainer(c.ID)
	assert.NoError(err)

	// Verify that we no longer find it.
	c, err = dockerutil.FindContainerByName(containerName)
	assert.NoError(err)
	assert.Nil(c)
}

func TestGetContainerEnv(t *testing.T) {
	assert := asrt.New(t)

	c, err := dockerutil.FindContainerByLabels(map[string]string{
		"com.ddev.site-name":        testContainerName,
		"com.docker.compose.oneoff": "False",
	})
	assert.NoError(err)
	require.NotEmpty(t, c)

	env := dockerutil.GetContainerEnv("HOTDOG", *c)
	assert.Equal("superior-to-corndog", env)
	env = dockerutil.GetContainerEnv("POTATO", *c)
	assert.Equal("future-fry", env)
	env = dockerutil.GetContainerEnv("NONEXISTENT", *c)
	assert.Equal("", env)
}

// TestRunSimpleContainer does a simple test of RunSimpleContainer()
func TestRunSimpleContainer(t *testing.T) {
	basename := fileutil.RandomFilenameBase()
	pwd, _ := os.Getwd()
	pwd, _ = filepath.Abs(pwd)
	testdata := filepath.Join(pwd, "testdata")
	require.DirExists(t, testdata)

	// Try the success case; script found, runs, all good.
	_, out, err := dockerutil.RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"//tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + "://tempmount"}, "25", true, false, nil, nil, nil)
	require.NoError(t, err)
	require.Contains(t, out, "simplescript.sh; TEMPENV=someenv UID=25")
	require.Contains(t, out, "stdout is captured")
	require.Contains(t, out, "stderr is also captured")

	// Try the case of running nonexistent script
	_, _, err = dockerutil.RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"nocommandbythatname"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + ":/tempmount"}, "25", true, false, nil, nil, nil)
	require.Error(t, err)
	if err != nil {
		require.Contains(t, err.Error(), "failed to StartContainer")
	}

	// Try the case of running a script that fails
	_, _, err = dockerutil.RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"/tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv", "ERROROUT=true"}, []string{testdata + ":/tempmount"}, "25", true, false, nil, nil, nil)
	require.Error(t, err)
	if err != nil {
		require.Contains(t, err.Error(), "container run failed with exit code 5")
	}

	// Provide an unqualified tag name
	_, _, err = dockerutil.RunSimpleContainer("busybox", "TestRunSimpleContainer"+basename, nil, nil, nil, nil, "", true, false, nil, nil, nil)
	require.Error(t, err)
	if err != nil {
		require.Contains(t, err.Error(), "image name must specify tag")
	}

	// Provide a malformed tag name
	_, _, err = dockerutil.RunSimpleContainer("busybox:", "TestRunSimpleContainer"+basename, nil, nil, nil, nil, "", true, false, nil, nil, nil)
	require.Error(t, err)
	if err != nil {
		require.Contains(t, err.Error(), "malformed tag provided")
	}
}

// TestGetBoundHostPorts() checks to see that the ports expected
// to be exposed on a container actually are exposed.
func TestGetBoundHostPorts(t *testing.T) {
	assert := asrt.New(t)

	testContainer, err := dockerutil.FindContainerByLabels(map[string]string{
		"com.ddev.site-name":        testContainerName,
		"com.docker.compose.oneoff": "False",
	})
	require.NoError(t, err)
	require.NotNil(t, testContainer)
	ports, err := dockerutil.GetBoundHostPorts(testContainer.ID)
	assert.NoError(err)
	assert.NotNil(ports)
	assert.Equal([]string{"8889", "8890"}, ports)
}

// TestGetRouterNetworkAliases() checks that network aliases can be retrieved from a container
func TestGetRouterNetworkAliases(t *testing.T) {
	assert := asrt.New(t)

	// Test with invalid container ID - should return error
	_, err := dockerutil.GetRouterNetworkAliases("nonexistent-container-id")
	assert.Error(err)

	// Test with a real container - should not error
	// (aliases may be empty if container isn't on ddev_default network)
	testContainer, err := dockerutil.FindContainerByLabels(map[string]string{
		"com.ddev.site-name":        testContainerName,
		"com.docker.compose.oneoff": "False",
	})
	require.NoError(t, err)
	require.NotNil(t, testContainer)
	aliases, err := dockerutil.GetRouterNetworkAliases(testContainer.ID)
	assert.NoError(err)
	assert.NotNil(aliases)
	// The test container may or may not have network aliases, so we just verify
	// the function returns without error and returns a valid (possibly empty) slice
}

// TestDockerExec() checks docker.Exec()
func TestDockerExec(t *testing.T) {
	assert := asrt.New(t)
	ctx, apiClient, err := dockerutil.GetDockerClient()
	if err != nil {
		t.Fatalf("Could not get docker client: %v", err)
	}

	id, _, err := dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, "", []string{"tail", "-f", "/dev/null"}, nil, nil, nil, "0", false, true, nil, nil, nil)
	assert.NoError(err)

	t.Cleanup(func() {
		_, err = apiClient.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
		assert.NoError(err)
	})

	stdout, _, err := dockerutil.Exec(id, "ls /etc", "")
	assert.NoError(err)
	assert.Contains(stdout, "group\nhostname")

	_, stderr, err := dockerutil.Exec(id, "ls /nothingthere", "")
	assert.Error(err)
	assert.Contains(stderr, "No such file or directory")
}

// TestCopyIntoContainer makes sure CopyIntoContainer copies a local file or directory into a specified
// path in container
func TestCopyIntoContainer(t *testing.T) {
	assert := asrt.New(t)
	pwd, _ := os.Getwd()

	cid, err := dockerutil.FindContainerByName(testContainerName)
	require.NoError(t, err)
	require.NotNil(t, cid)

	uid, _, _ := dockerutil.GetContainerUser()
	targetDir, _, err := dockerutil.Exec(cid.ID, "mktemp -d", uid)
	require.NoError(t, err)
	targetDir = strings.Trim(targetDir, "\n")

	err = dockerutil.CopyIntoContainer(filepath.Join(pwd, "testdata", t.Name()), testContainerName, targetDir, "")
	require.NoError(t, err)

	out, _, err := dockerutil.Exec(cid.ID, fmt.Sprintf(`bash -c "cd %s && ls -R * .test.sh && ./.test.sh"`, targetDir), uid)
	require.NoError(t, err)
	assert.Equal(`.test.sh
root.txt

subdir1:
subdir1.txt
hi this is a test file
`, out)

	// Now try a file
	err = dockerutil.CopyIntoContainer(filepath.Join(pwd, "testdata", t.Name(), "root.txt"), testContainerName, "/tmp", "")
	require.NoError(t, err)

	out, _, err = dockerutil.Exec(cid.ID, `cat /tmp/root.txt`, uid)
	require.NoError(t, err)
	assert.Equal("root.txt here\n", out)
}

// TestCopyFromContainer makes sure CopyFromContainer copies a container into a specified
// local directory
func TestCopyFromContainer(t *testing.T) {
	assert := asrt.New(t)
	containerSourceDir := "/var/tmp/backdrop_drush_commands/backdrop-drush-extension"
	containerExpectedFile := "backdrop.drush.inc"
	cid, err := dockerutil.FindContainerByName(testContainerName)
	require.NoError(t, err)
	require.NotNil(t, cid)

	targetDir := testcommon.CreateTmpDir(t.Name())
	require.NoError(t, err)

	err = dockerutil.CopyFromContainer(testContainerName, containerSourceDir, targetDir)
	require.NoError(t, err)

	assert.FileExists(filepath.Join(targetDir, path.Base(containerSourceDir), containerExpectedFile))
}
