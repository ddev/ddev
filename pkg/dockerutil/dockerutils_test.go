package dockerutil_test

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerFilters "github.com/docker/docker/api/types/filters"
	dockerVolume "github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	logOutput "github.com/sirupsen/logrus"
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
	output.LogSetUp()

	_ = os.Setenv("DDEV_NONINTERACTIVE", "true")
	_ = os.Setenv("MUTAGEN_DATA_DIRECTORY", globalconfig.GetMutagenDataDirectory())

	labels := map[string]string{
		"com.ddev.site-name": testContainerName,
	}

	// Prep Docker container for Docker util tests
	imageExists, err := dockerutil.ImageExistsLocally(ddevImages.GetWebImage())
	if err != nil {
		logOutput.Errorf("Failed to check for local image %s: %v", ddevImages.GetWebImage(), err)
		return 6
	}
	if !imageExists {
		err := dockerutil.Pull(ddevImages.GetWebImage())
		if err != nil {
			logOutput.Errorf("Failed to pull test image: %v", err)
			return 7
		}
	}

	foundContainer, _ := dockerutil.FindContainerByLabels(labels)

	if foundContainer != nil {
		err = dockerutil.RemoveContainer(foundContainer.ID)
		if err != nil {
			logOutput.Errorf("-- FAIL: dockerutils_test TestMain failed to remove container %s: %v", foundContainer.ID, err)
			return 5
		}
	}

	containerID, err := startTestContainer()
	if err != nil {
		logOutput.Errorf("-- FAIL: dockerutils_test failed to start test container: %v", err)
		return 3
	}
	defer func() {
		foundContainer, _ := dockerutil.FindContainerByLabels(labels)
		if foundContainer != nil {
			err = dockerutil.RemoveContainer(foundContainer.ID)
			if err != nil {
				logOutput.Errorf("-- FAIL: dockerutils_test failed to remove test container: %v", err)
			}
		}
	}()

	log.Printf("ContainerWait at %v", time.Now())
	out, err := dockerutil.ContainerWait(60, map[string]string{"com.ddev.site-name": testContainerName})
	log.Printf("ContainerWait returrned at %v out='%s' err='%v'", time.Now(), out, err)

	if err != nil {
		logout, _ := exec.RunHostCommand("docker", "logs", containerID)
		inspectOut, _ := exec.RunHostCommand("sh", "-c", fmt.Sprintf("docker inspect %s|jq -r '.[0].State.Health.Log'", containerID))
		log.Printf("FAIL: dockerutils_test testMain failed to ContainerWait for container: %v, logs\n========= container logs ======\n%s\n======= end logs =======\n==== health log =====\ninspectOut\n%s\n========", err, logout, inspectOut)
		return 4
	}
	exitStatus := m.Run()

	return exitStatus
}

// Start container for tests; returns containerID, error
func startTestContainer() (string, error) {
	portBinding := map[nat.Port][]nat.PortBinding{
		"80/tcp": {
			{HostPort: "8889"},
		},
		"8025/tcp": {
			{HostPort: "8890"},
		}}
	containerID, _, err := dockerutil.RunSimpleContainer(ddevImages.GetWebImage(), testContainerName, nil, nil, []string{
		"HOTDOG=superior-to-corndog",
		"POTATO=future-fry",
		"DDEV_WEBSERVER_TYPE=nginx-fpm",
	}, nil, "33", false, true, map[string]string{"com.docker.compose.service": "web", "com.ddev.site-name": testContainerName}, portBinding, nil)
	if err != nil {
		return "", err
	}

	return containerID, nil
}

// TestGetContainerHealth tests the function for processing container readiness.
func TestGetContainerHealth(t *testing.T) {
	assert := asrt.New(t)
	ctx, client := dockerutil.GetDockerClient()

	labels := map[string]string{
		"com.ddev.site-name": testContainerName,
	}

	t.Cleanup(func() {
		container, err := dockerutil.FindContainerByLabels(labels)
		if err == nil || container != nil {
			err = dockerutil.RemoveContainer(container.ID)
			assert.NoError(err)
		}

		// Make sure test container exists again
		_, err = startTestContainer()
		assert.NoError(err)
		healthDetail, err := dockerutil.ContainerWait(30, labels)
		assert.NoError(err, "healthDetail='%s'", healthDetail)

		container, err = dockerutil.FindContainerByLabels(labels)
		assert.NoError(err)
		assert.NotNil(container)

		status, healthDetail := dockerutil.GetContainerHealth(container)
		assert.Contains(healthDetail, "/var/www/html:OK mailpit:OK phpstatus:OK ")
		assert.Equal("healthy", status)
	})

	container, err := dockerutil.FindContainerByLabels(labels)
	require.NoError(t, err)
	require.NotNil(t, container)

	status, log := dockerutil.GetContainerHealth(container)
	assert.Equal("healthy", status, "container should be healthy; log=%v", log)

	// Now break the container and make sure it's unhealthy
	timeout := 10
	err = client.ContainerStop(ctx, container.ID, dockerContainer.StopOptions{Timeout: &timeout})
	assert.NoError(err)

	status, log = dockerutil.GetContainerHealth(container)
	assert.Equal("unhealthy", status, "container should be unhealthy; log=%v", log)
	assert.NoError(err)

}

// TestContainerWait tests the error cases for the container check wait loop.
func TestContainerWait(t *testing.T) {
	assert := asrt.New(t)

	labels := map[string]string{
		"com.ddev.site-name": testContainerName,
	}

	// Try a zero-wait, should show timed-out
	_, err := dockerutil.ContainerWait(0, labels)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "health check timed out")
	}

	// Try 30-second wait for "healthy", should show OK
	healthDetail, err := dockerutil.ContainerWait(30, labels)
	assert.NoError(err)

	assert.Contains(healthDetail, "phpstatus:OK")

	// Try a nonexistent container, should get error
	labels = map[string]string{"com.ddev.site-name": "nothing-there"}
	_, err = dockerutil.ContainerWait(1, labels)
	require.Error(t, err)
	assert.Contains(err.Error(), "failed to query container")

	// If we run a quick container and it immediately exits, ContainerWait should find it is not there
	// and note that it exited.
	labels = map[string]string{"test": "quickexit"}
	_ = dockerutil.RemoveContainersByLabels(labels)
	cID, _, err := dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, t.Name()+util.RandString(5), []string{"ls"}, nil, nil, nil, "0", false, true, labels, nil, nil)
	t.Cleanup(func() {
		_ = dockerutil.RemoveContainer(cID)
	})
	require.NoError(t, err)
	_, err = dockerutil.ContainerWait(5, labels)
	assert.Error(err)
	assert.Contains(err.Error(), "container exited")
	_ = dockerutil.RemoveContainer(cID)

	// If we run a container that does not have a healthcheck
	// it should be found as good immediately
	labels = map[string]string{"test": "nohealthcheck"}
	_ = dockerutil.RemoveContainersByLabels(labels)
	cID, _, err = dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, t.Name()+util.RandString(5), []string{"sleep", "60"}, nil, nil, nil, "0", false, true, labels, nil, nil)
	t.Cleanup(func() {
		_ = dockerutil.RemoveContainer(cID)
	})
	require.NoError(t, err)
	_, err = dockerutil.ContainerWait(5, labels)
	assert.NoError(err)

	ddevWebserver := versionconstants.WebImg + ":" + versionconstants.WebTag
	// If we run a container that *does* have a healthcheck but it's unhealthy
	// then ContainerWait shouldn't return until specified wait, and should fail
	// Use ddev-webserver for this; it won't have good health on normal run
	labels = map[string]string{"test": "hashealthcheckbutbad"}
	_ = dockerutil.RemoveContainersByLabels(labels)
	cID, _, err = dockerutil.RunSimpleContainer(ddevWebserver, t.Name()+util.RandString(5), []string{"sleep", "5"}, nil, []string{"DDEV_WEBSERVER_TYPE=nginx-fpm"}, nil, "0", false, true, labels, nil, nil)
	t.Cleanup(func() {
		_ = dockerutil.RemoveContainer(cID)
	})
	require.NoError(t, err)
	_, err = dockerutil.ContainerWait(3, labels)
	assert.Error(err)
	assert.Contains(err.Error(), "timed out without becoming healthy")
	_ = dockerutil.RemoveContainer(cID)

	// If we run a container that *does* have a healthcheck but it's not healthy for a while
	// then ContainerWait should detect failure early, but should succeed later
	labels = map[string]string{"test": "hashealthcheckbutbad"}
	_ = dockerutil.RemoveContainersByLabels(labels)
	cID, _, err = dockerutil.RunSimpleContainer(ddevWebserver, t.Name()+util.RandString(5), []string{"bash", "-c", "sleep 5 && /start.sh"}, nil, []string{"DDEV_WEBSERVER_TYPE=nginx-fpm"}, nil, "0", false, true, labels, nil, nil)
	t.Cleanup(func() {
		_ = dockerutil.RemoveContainer(cID)
	})
	require.NoError(t, err)
	_, err = dockerutil.ContainerWait(3, labels)
	assert.Error(err)
	assert.Contains(err.Error(), "timed out without becoming healthy")
	// Try it again, wait 60s for health; on macOS it usually takes about 2s for ddev-webserver to become healthy
	out, err := dockerutil.ContainerWait(60, labels)
	assert.NoError(err, "output=%s", out)
}

// TestComposeCmd tests execution of docker-compose commands.
func TestComposeCmd(t *testing.T) {
	assert := asrt.New(t)

	composeFiles := []string{filepath.Join("testdata", "docker-compose.yml")}

	stdout, stderr, err := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: composeFiles,
		Action:       []string{"config", "--services"},
	})
	assert.NoError(err)
	assert.Contains(stdout, "web")
	assert.Contains(stdout, "db")
	assert.Contains(stderr, "Defaulting to a blank string")

	composeFiles = append(composeFiles, filepath.Join("testdata", "docker-compose.override.yml"))

	stdout, stderr, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: composeFiles,
		Action:       []string{"config", "--services"},
	})
	assert.NoError(err)
	assert.Contains(stdout, "web")
	assert.Contains(stdout, "db")
	assert.Contains(stdout, "foo")
	assert.Contains(stderr, "Defaulting to a blank string")

	composeFiles = []string{"invalid.yml"}
	_, _, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: composeFiles,
		Action:       []string{"config", "--services"},
	})
	assert.Error(err)
}

// TestComposeWithStreams tests execution of docker-compose commands with streams
func TestComposeWithStreams(t *testing.T) {
	assert := asrt.New(t)

	container, _ := dockerutil.FindContainerByName(t.Name())
	if container != nil {
		_ = dockerutil.RemoveContainer(container.ID)
	}

	// Use the current actual web container for this, so replace in base docker-compose file
	composeBase := filepath.Join("testdata", "TestComposeWithStreams", "test-compose-with-streams.yaml")
	tmp, err := os.MkdirTemp("", "")
	assert.NoError(err)
	realComposeFile := filepath.Join(tmp, "replaced-compose-with-streams.yaml")

	err = fileutil.ReplaceStringInFile("TEST-COMPOSE-WITH-STREAMS-IMAGE", versionconstants.WebImg+":"+versionconstants.WebTag, composeBase, realComposeFile)
	assert.NoError(err)

	composeFiles := []string{realComposeFile}

	t.Cleanup(func() {
		_, _, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
			ComposeFiles: composeFiles,
			Action:       []string{"down"},
		})
		assert.NoError(err)
	})

	_, _, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: composeFiles,
		Action:       []string{"up", "-d"},
	})
	require.NoError(t, err)

	_, err = dockerutil.ContainerWait(60, map[string]string{"com.ddev.site-name": t.Name()})
	if err != nil {
		logout, _ := exec.RunCommand("docker", []string{"logs", t.Name()})
		inspectOut, _ := exec.RunCommandPipe("sh", []string{"-c", fmt.Sprintf("docker inspect %s|jq -r '.[0].State.Health.Log'", t.Name())})
		t.Fatalf("FAIL: TestComposeWithStreams failed to ContainerWait for container: %v, logs\n========= container logs ======\n%s\n======= end logs =======\n==== health log =====\ninspectOut\n%s\n========", err, logout, inspectOut)
	}

	// Point stdout to os.Stdout and do simple ps -ef in web container
	stdout := util.CaptureStdOut()
	err = dockerutil.ComposeWithStreams(composeFiles, os.Stdin, os.Stdout, os.Stderr, "exec", "-T", "web", "ps", "-ef")
	assert.NoError(err)
	output := stdout()
	assert.Contains(output, "supervisord")

	// Reverse stdout and stderr and create an error and normal stdout. We should see only the error captured in stdout
	stdout = util.CaptureStdOut()
	err = dockerutil.ComposeWithStreams(composeFiles, os.Stdin, os.Stderr, os.Stdout, "exec", "-T", "web", "ls", "-d", "xx", "/var/run/apache2")
	assert.Error(err)
	output = stdout()
	assert.Contains(output, "ls: cannot access 'xx': No such file or directory")

	// Flip stdout and stderr and create an error and normal stdout. We should see only the success captured in stdout
	stdout = util.CaptureStdOut()
	err = dockerutil.ComposeWithStreams(composeFiles, os.Stdin, os.Stdout, os.Stderr, "exec", "-T", "web", "ls", "-d", "xx", "/var/run/apache2")
	assert.Error(err)
	output = stdout()
	assert.Contains(output, "/var/run/apache2", output)
}

// TestCheckCompose tests detection of docker-compose.
func TestCheckCompose(t *testing.T) {
	assert := asrt.New(t)

	globalconfig.DockerComposeVersion = ""
	composeErr := dockerutil.CheckDockerCompose()
	if composeErr != nil {
		out, err := exec.RunHostCommand(DdevBin, "config", "global")
		require.NoError(t, err)
		ddevVersion, err := exec.RunHostCommand(DdevBin, "version")
		require.NoError(t, err)
		assert.NoError(composeErr, "RequiredDockerComposeVersion=%s global config=%s ddevVersion=%s", globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion, out, ddevVersion)
	}
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
	cID, _, err := dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, containerName, []string{"//tempmount/sleepALittle.sh"}, nil, nil, []string{testdata + "://tempmount"}, "25", false, false, nil, nil, nil)
	assert.NoError(err)

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

	container, err := dockerutil.FindContainerByLabels(map[string]string{"com.ddev.site-name": testContainerName})
	assert.NoError(err)
	require.NotEmpty(t, container)

	env := dockerutil.GetContainerEnv("HOTDOG", *container)
	assert.Equal("superior-to-corndog", env)
	env = dockerutil.GetContainerEnv("POTATO", *container)
	assert.Equal("future-fry", env)
	env = dockerutil.GetContainerEnv("NONEXISTENT", *container)
	assert.Equal("", env)
}

// TestRunSimpleContainer does a simple test of RunSimpleContainer()
func TestRunSimpleContainer(t *testing.T) {
	assert := asrt.New(t)

	basename := fileutil.RandomFilenameBase()
	pwd, _ := os.Getwd()
	pwd, _ = filepath.Abs(pwd)
	testdata := filepath.Join(pwd, "testdata")
	assert.DirExists(testdata)

	// Try the success case; script found, runs, all good.
	_, out, err := dockerutil.RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"//tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + "://tempmount"}, "25", true, false, nil, nil, nil)
	assert.NoError(err)
	assert.Contains(out, "simplescript.sh; TEMPENV=someenv UID=25")
	assert.Contains(out, "stdout is captured")
	assert.Contains(out, "stderr is also captured")

	// Try the case of running nonexistent script
	_, _, err = dockerutil.RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"nocommandbythatname"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + ":/tempmount"}, "25", true, false, nil, nil, nil)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "failed to StartContainer")
	}

	// Try the case of running a script that fails
	_, _, err = dockerutil.RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"/tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv", "ERROROUT=true"}, []string{testdata + ":/tempmount"}, "25", true, false, nil, nil, nil)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "container run failed with exit code 5")
	}

	// Provide an unqualified tag name
	_, _, err = dockerutil.RunSimpleContainer("busybox", "TestRunSimpleContainer"+basename, nil, nil, nil, nil, "", true, false, nil, nil, nil)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "image name must specify tag")
	}

	// Provide a malformed tag name
	_, _, err = dockerutil.RunSimpleContainer("busybox:", "TestRunSimpleContainer"+basename, nil, nil, nil, nil, "", true, false, nil, nil, nil)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "malformed tag provided")
	}
}

// TestGetExposedContainerPorts() checks to see that the ports expected
// to be exposed on a container actually are exposed.
func TestGetExposedContainerPorts(t *testing.T) {
	assert := asrt.New(t)

	testContainer, err := dockerutil.FindContainerByLabels(map[string]string{"com.ddev.site-name": testContainerName})
	require.NoError(t, err)
	require.NotNil(t, testContainer)
	ports, err := dockerutil.GetExposedContainerPorts(testContainer.ID)
	assert.NoError(err)
	assert.NotNil(ports)
	assert.Equal([]string{"8889", "8890"}, ports)
}

// TestDockerExec() checks docker.Exec()
func TestDockerExec(t *testing.T) {
	assert := asrt.New(t)
	ctx, client := dockerutil.GetDockerClient()

	id, _, err := dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"tail", "-f", "/dev/null"}, nil, nil, nil, "0", false, true, nil, nil, nil)
	assert.NoError(err)

	t.Cleanup(func() {
		err = client.ContainerRemove(ctx, id, dockerContainer.RemoveOptions{Force: true})
		assert.NoError(err)
	})

	stdout, _, err := dockerutil.Exec(id, "ls /etc", "")
	assert.NoError(err)
	assert.Contains(stdout, "group\nhostname")

	_, stderr, err := dockerutil.Exec(id, "ls /nothingthere", "")
	assert.Error(err)
	assert.Contains(stderr, "No such file or directory")

}

// TestCreateVolume does a trivial test of creating a trivial Docker volume.
func TestCreateVolume(t *testing.T) {
	assert := asrt.New(t)
	// Make sure there's no existing volume.
	//nolint: errcheck
	dockerutil.RemoveVolume("junker99")
	volume, err := dockerutil.CreateVolume("junker99", "local", map[string]string{}, nil)
	require.NoError(t, err)

	//nolint: errcheck
	defer dockerutil.RemoveVolume("junker99")
	require.NotNil(t, volume)
	assert.Equal("junker99", volume.Name)
}

// TestRemoveVolume makes sure we can remove a volume successfully
func TestRemoveVolume(t *testing.T) {
	assert := asrt.New(t)
	ctx, client := dockerutil.GetDockerClient()

	testVolume := "junker999"
	spareVolume := "someVolumeThatCanNeverExit"

	_ = dockerutil.RemoveVolume(testVolume)
	pwd, err := os.Getwd()
	assert.NoError(err)

	source := pwd
	if runtime.GOOS == "darwin" && fileutil.IsDirectory(filepath.Join("/System/Volumes/Data", source)) {
		source = filepath.Join("/System/Volumes/Data", source)
	}
	nfsServerAddr, _ := dockerutil.GetNFSServerAddr()
	volume, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{"type": "nfs", "o": fmt.Sprintf("addr=%s,hard,nolock,rw,wsize=32768,rsize=32768", nfsServerAddr),
		"device": ":" + dockerutil.MassageWindowsNFSMount(source)}, nil)
	assert.NoError(err)

	volumes, err := client.VolumeList(ctx, dockerVolume.ListOptions{Filters: dockerFilters.NewArgs(dockerFilters.KeyValuePair{Key: "name", Value: testVolume})})
	assert.NoError(err)
	require.Len(t, volumes.Volumes, 1)
	assert.Equal(testVolume, volumes.Volumes[0].Name)

	require.NotNil(t, volume)
	assert.Equal(testVolume, volume.Name)
	err = dockerutil.RemoveVolume(testVolume)
	assert.NoError(err)

	volumes, err = client.VolumeList(ctx, dockerVolume.ListOptions{Filters: dockerFilters.NewArgs(dockerFilters.KeyValuePair{Key: "name", Value: testVolume})})
	assert.NoError(err)
	assert.Empty(volumes.Volumes)

	// Make sure spareVolume doesn't exist, then make sure removal
	// of nonexistent volume doesn't result in error
	_ = dockerutil.RemoveVolume(spareVolume)
	err = dockerutil.RemoveVolume(spareVolume)
	assert.NoError(err)

}

// TestCopyIntoVolume makes sure CopyToVolume copies a local directory into a volume
func TestCopyIntoVolume(t *testing.T) {
	assert := asrt.New(t)
	err := dockerutil.RemoveVolume(t.Name())
	assert.NoError(err)

	pwd, _ := os.Getwd()
	t.Cleanup(func() {
		err = dockerutil.RemoveVolume(t.Name())
		assert.NoError(err)
	})

	err = dockerutil.CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name()), t.Name(), "", "0", "", true)
	require.NoError(t, err)

	// Make sure that the content is the same, and that .test.sh is executable
	// On Windows the upload can result in losing executable bit
	_, out, err := dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"sh", "-c", "cd /mnt/" + t.Name() + " && ls -R .test.sh * && ./.test.sh"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil, nil, nil)
	assert.NoError(err)
	assert.Equal(`.test.sh
root.txt

subdir1:
subdir1.txt
hi this is a test file
`, out)

	err = dockerutil.CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name()), t.Name(), "somesubdir", "501", "", true)
	assert.NoError(err)
	_, out, err = dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"sh", "-c", "cd /mnt/" + t.Name() + "/somesubdir  && pwd && ls -R"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "0", true, false, nil, nil, nil)
	assert.NoError(err)
	assert.Equal(`/mnt/TestCopyIntoVolume/somesubdir
.:
root.txt
subdir1

./subdir1:
subdir1.txt
`, out)

	// Now try a file
	err = dockerutil.CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name(), "root.txt"), t.Name(), "", "0", "", true)
	assert.NoError(err)

	// Make sure that the content is the same, and that .test.sh is executable
	_, out, err = dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"cat", "/mnt/" + t.Name() + "/root.txt"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil, nil, nil)
	assert.NoError(err)
	assert.Equal("root.txt here\n", out)

	// Copy destructively and make sure that stuff got destroyed
	err = dockerutil.CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name()+"2"), t.Name(), "", "0", "", true)
	require.NoError(t, err)

	_, _, err = dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"ls", "/mnt/" + t.Name() + "/subdir1/subdir1.txt"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil, nil, nil)
	require.Error(t, err)

	_, _, err = dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"ls", "/mnt/" + t.Name() + "/subdir1/only-the-new-stuff.txt"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil, nil, nil)
	require.NoError(t, err)
}

// TestDockerIP tries out a number of DOCKER_HOST permutations
// to verify that GetDockerIP does them right
func TestGetDockerIP(t *testing.T) {
	assert := asrt.New(t)

	expectations := map[string]string{
		"":                            "127.0.0.1",
		"unix:///var/run/docker.sock": "127.0.0.1",
		"unix:///Users/rfay/.docker/run/docker.sock": "127.0.0.1",
		"unix:///Users/rfay/.colima/docker.sock":     "127.0.0.1",
		"tcp://185.199.110.153:2375":                 "185.199.110.153",
	}

	for k, v := range expectations {
		t.Setenv("DOCKER_HOST", k)
		// DockerIP is cached, so we have to reset it to check
		dockerutil.DockerIP = ""
		result, err := dockerutil.GetDockerIP()
		assert.NoError(err)
		assert.Equal(v, result, "for %s expected %s, got %s", k, v, result)
	}
}

// TestCopyIntoContainer makes sure CopyIntoContainer copies a local file or directory into a specified
// path in container
func TestCopyIntoContainer(t *testing.T) {
	assert := asrt.New(t)
	pwd, _ := os.Getwd()

	cid, err := dockerutil.FindContainerByName(testContainerName)
	require.NoError(t, err)
	require.NotNil(t, cid)

	uid, _, _ := util.GetContainerUIDGid()
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

	targetDir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)

	err = dockerutil.CopyFromContainer(testContainerName, containerSourceDir, targetDir)
	require.NoError(t, err)

	assert.FileExists(filepath.Join(targetDir, path.Base(containerSourceDir), containerExpectedFile))
}
