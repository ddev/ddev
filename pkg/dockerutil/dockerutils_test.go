package dockerutil_test

import (
	"fmt"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/versionconstants"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	logOutput "github.com/sirupsen/logrus"

	"path/filepath"

	. "github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/fsouza/go-dockerclient"
	asrt "github.com/stretchr/testify/assert"
)

var testContainerName = "TestDockerUtils"

func TestMain(m *testing.M) { os.Exit(testMain(m)) }

func testMain(m *testing.M) int {
	output.LogSetUp()

	EnsureDdevNetwork()

	_ = os.Setenv("DDEV_NONINTERACTIVE", "true")

	// prep docker container for docker util tests
	client := GetDockerClient()
	imageExists, err := ImageExistsLocally(versionconstants.WebImg + ":" + versionconstants.WebTag)
	if err != nil {
		logOutput.Errorf("Failed to check for local image %s: %v", versionconstants.WebImg+":"+versionconstants.WebTag, err)
		return 6
	}
	if !imageExists {
		err := client.PullImage(docker.PullImageOptions{
			Repository: versionconstants.WebImg,
			Tag:        versionconstants.WebTag,
		}, docker.AuthConfiguration{})
		if err != nil {
			logOutput.Errorf("failed to pull test image: %v", err)
			return 7
		}
	}

	foundContainer, _ := FindContainerByLabels(map[string]string{"com.ddev.site-name": testContainerName})

	if foundContainer != nil {
		err = RemoveContainer(foundContainer.ID, 30)
		if err != nil {
			logOutput.Errorf("-- FAIL: dockerutils_test TestMain failed to remove container %s: %v", foundContainer.ID, err)
			return 5
		}
	}

	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Name: testContainerName,
		Config: &docker.Config{
			Image: versionconstants.WebImg + ":" + versionconstants.WebTag,
			Labels: map[string]string{
				"com.docker.compose.service": "web",
				"com.ddev.site-name":         testContainerName,
			},
			Env:  []string{"HOTDOG=superior-to-corndog", "POTATO=future-fry"},
			User: "33:33", // The "www-data" pre-installed in container
		},
		HostConfig: &docker.HostConfig{
			PortBindings: map[docker.Port][]docker.PortBinding{
				"80/tcp": {
					{HostPort: "8889"},
				},
				"8025/tcp": {
					{HostPort: "8890"},
				},
			},
		},
	})
	if err != nil {
		logOutput.Errorf("failed to create/start docker container: %v", err)
		return 1
	}
	err = client.StartContainer(container.ID, nil)
	if err != nil {
		logOutput.Errorf("-- FAIL: dockerutils_test failed to StartContainer: %v", err)
		return 2
	}
	defer func() {
		err = RemoveContainer(container.ID, 10)
		if err != nil {
			logOutput.Errorf("-- FAIL: dockerutils_test failed to remove test container: %v", err)
		}
	}()
	_, err = ContainerWait(30, map[string]string{"com.ddev.site-name": testContainerName})
	if err != nil {
		logout, _ := exec.RunCommand("docker", []string{"logs", container.Name})
		inspectOut, _ := exec.RunCommandPipe("sh", []string{"-c", fmt.Sprintf("docker inspect %s|jq -r '.[0].State.Health.Log'", container.Name)})
		_ = fmt.Errorf("FAIL: dockerutils_test failed to ContainerWait for container: %v, logs\n========= container logs ======\n%s\n======= end logs =======\n==== health log =====\ninspectOut\n%s\n========", err, logout, inspectOut)
		return 4
	}
	exitStatus := m.Run()

	return exitStatus
}

// TestGetContainerHealth tests the function for processing container readiness.
func TestGetContainerHealth(t *testing.T) {
	assert := asrt.New(t)
	client := GetDockerClient()

	labels := map[string]string{
		"com.ddev.site-name": testContainerName,
	}
	container, err := FindContainerByLabels(labels)
	require.NoError(t, err)
	require.NotNil(t, container)

	err = client.StopContainer(container.ID, 10)
	assert.NoError(err)

	status, _ := GetContainerHealth(container)
	assert.Equal(status, "unhealthy")

	err = client.StartContainer(container.ID, nil)
	assert.NoError(err)
	healthDetail, err := ContainerWait(30, labels)
	assert.NoError(err)

	assert.Equal("phpstatus: OK /var/www/html: OK mailhog: OK ", healthDetail)

	status, healthDetail = GetContainerHealth(container)
	assert.Equal("healthy", status)
	assert.Equal("phpstatus: OK /var/www/html: OK mailhog: OK ", healthDetail)
}

// TestContainerWait tests the error cases for the container check wait loop.
func TestContainerWait(t *testing.T) {
	if IsColima() {
		t.Skip("Skipping on colima because of so many odd failures")
	}
	assert := asrt.New(t)

	labels := map[string]string{
		"com.ddev.site-name": testContainerName,
	}

	// Try a zero-wait, should show timed-out
	_, err := ContainerWait(0, labels)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "health check timed out")
	}

	// Try 15-second wait for "healthy", should show OK
	healthDetail, err := ContainerWait(30, labels)
	assert.NoError(err)

	assert.Contains(healthDetail, "phpstatus: OK")

	// Try a nonexistent container, should get error
	labels = map[string]string{"com.ddev.site-name": "nothing-there"}
	_, err = ContainerWait(1, labels)
	require.Error(t, err)
	assert.Contains(err.Error(), "failed to query container")

	// If we just run a quick container and it immediately exits, ContainerWait should find it not there
	// and note that it exited.
	labels = map[string]string{"test": "quickexit"}
	_ = RemoveContainersByLabels(labels)
	cID, _, err := RunSimpleContainer(versionconstants.BusyboxImage, t.Name()+util.RandString(5), []string{"ls"}, nil, nil, nil, "0", false, true, labels)
	t.Cleanup(func() {
		_ = RemoveContainer(cID, 0)
	})
	require.NoError(t, err)
	_, err = ContainerWait(5, labels)
	assert.Error(err)
	assert.Contains(err.Error(), "container exited")
	_ = RemoveContainer(cID, 0)

	// If we run a container that does not have a healthcheck
	// it should be found as good immediately
	labels = map[string]string{"test": "nohealthcheck"}
	_ = RemoveContainersByLabels(labels)
	cID, _, err = RunSimpleContainer(versionconstants.BusyboxImage, t.Name()+util.RandString(5), []string{"sleep", "60"}, nil, nil, nil, "0", false, true, labels)
	t.Cleanup(func() {
		_ = RemoveContainer(cID, 0)
	})
	require.NoError(t, err)
	_, err = ContainerWait(5, labels)
	assert.NoError(err)

	ddevWebserver := versionconstants.WebImg + ":" + versionconstants.WebTag
	// If we run a container that *does* have a healthcheck but it's unhealthy
	// then ContainerWait shouldn't return until specified wait, and should fail
	// Use ddev-webserver for this; it won't have good health on normal run
	labels = map[string]string{"test": "hashealthcheckbutbad"}
	_ = RemoveContainersByLabels(labels)
	cID, _, err = RunSimpleContainer(ddevWebserver, t.Name()+util.RandString(5), []string{"sleep", "5"}, nil, nil, nil, "0", false, true, labels)
	t.Cleanup(func() {
		_ = RemoveContainer(cID, 0)
	})
	require.NoError(t, err)
	_, err = ContainerWait(3, labels)
	assert.Error(err)
	assert.Contains(err.Error(), "timed out without becoming healthy")
	_ = RemoveContainer(cID, 0)

	// If we run a container that *does* have a healthcheck but it's not healthy for a while
	// then ContainerWait should detect failure early, but should succeed later
	labels = map[string]string{"test": "hashealthcheckbutbad"}
	_ = RemoveContainersByLabels(labels)
	cID, _, err = RunSimpleContainer(ddevWebserver, t.Name()+util.RandString(5), []string{"bash", "-c", "sleep 5 && /start.sh"}, nil, nil, nil, "0", false, true, labels)
	t.Cleanup(func() {
		_ = RemoveContainer(cID, 0)
	})
	require.NoError(t, err)
	_, err = ContainerWait(3, labels)
	assert.Error(err)
	assert.Contains(err.Error(), "timed out without becoming healthy")
	// Try it again, wait 60s for health; on macOS it usually takes about 2s for ddev-webserver to become healthy
	out, err := ContainerWait(60, labels)
	assert.NoError(err, "output=%s", out)
}

// TestComposeCmd tests execution of docker-compose commands.
func TestComposeCmd(t *testing.T) {
	assert := asrt.New(t)

	composeFiles := []string{filepath.Join("testdata", "docker-compose.yml")}

	stdout, stderr, err := ComposeCmd(composeFiles, "config", "--services")
	assert.NoError(err)
	assert.Contains(stdout, "web")
	assert.Contains(stdout, "db")
	assert.Contains(stderr, "Defaulting to a blank string")

	composeFiles = append(composeFiles, filepath.Join("testdata", "docker-compose.override.yml"))

	stdout, stderr, err = ComposeCmd(composeFiles, "config", "--services")
	assert.NoError(err)
	assert.Contains(stdout, "web")
	assert.Contains(stdout, "db")
	assert.Contains(stdout, "foo")
	assert.Contains(stderr, "Defaulting to a blank string")

	composeFiles = []string{"invalid.yml"}
	_, _, err = ComposeCmd(composeFiles, "config", "--services")
	assert.Error(err)
}

// TestComposeWithStreams tests execution of docker-compose commands with streams
func TestComposeWithStreams(t *testing.T) {
	assert := asrt.New(t)

	container, _ := FindContainerByName(t.Name())
	if container != nil {
		_ = RemoveContainer(container.ID, 20)
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
		_, _, err = ComposeCmd(composeFiles, "down")
		assert.NoError(err)
	})

	_, _, err = ComposeCmd(composeFiles, "up", "-d")
	require.NoError(t, err)

	_, err = ContainerWait(30, map[string]string{"com.ddev.site-name": t.Name()})
	if err != nil {
		logout, _ := exec.RunCommand("docker", []string{"logs", t.Name()})
		inspectOut, _ := exec.RunCommandPipe("sh", []string{"-c", fmt.Sprintf("docker inspect %s|jq -r '.[0].State.Health.Log'", t.Name())})
		t.Fatalf("FAIL: dockerutils_test failed to ContainerWait for container: %v, logs\n========= container logs ======\n%s\n======= end logs =======\n==== health log =====\ninspectOut\n%s\n========", err, logout, inspectOut)
	}

	// Point stdout to os.Stdout and do simple ps -ef in web container
	stdout := util.CaptureStdOut()
	err = ComposeWithStreams(composeFiles, os.Stdin, os.Stdout, os.Stderr, "exec", "-T", "web", "ps", "-ef")
	assert.NoError(err)
	output := stdout()
	assert.Contains(output, "supervisord")

	// Reverse stdout and stderr and create an error and normal stdout. We should see only the error captured in stdout
	stdout = util.CaptureStdOut()
	err = ComposeWithStreams(composeFiles, os.Stdin, os.Stderr, os.Stdout, "exec", "-T", "web", "ls", "-d", "xx", "/var/run/apache2")
	assert.Error(err)
	output = stdout()
	assert.Contains(output, "ls: cannot access 'xx': No such file or directory")

	// Flip stdout and stderr and create an error and normal stdout. We should see only the success captured in stdout
	stdout = util.CaptureStdOut()
	err = ComposeWithStreams(composeFiles, os.Stdin, os.Stdout, os.Stderr, "exec", "-T", "web", "ls", "-d", "xx", "/var/run/apache2")
	assert.Error(err)
	output = stdout()
	assert.Contains(output, "/var/run/apache2", output)
}

// TestCheckCompose tests detection of docker-compose.
func TestCheckCompose(t *testing.T) {
	assert := asrt.New(t)

	err := CheckDockerCompose()
	assert.NoError(err)
}

// TestGetAppContainers looks for container with sitename dockerutils-test
func TestGetAppContainers(t *testing.T) {
	assert := asrt.New(t)
	containers, err := GetAppContainers(testContainerName)
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
	c, err := FindContainerByName(containerName)
	assert.NoError(err)
	if err == nil && c != nil {
		err = RemoveContainer(c.ID, 0)
		assert.NoError(err)
	}

	// Run a container, don't remove it.
	cID, _, err := RunSimpleContainer(versionconstants.BusyboxImage, containerName, []string{"//tempmount/sleepALittle.sh"}, nil, nil, []string{testdata + "://tempmount"}, "25", false, false, nil)
	assert.NoError(err)

	defer func() {
		_ = RemoveContainer(cID, 0)
	}()

	// Now find the container by name
	c, err = FindContainerByName(containerName)
	assert.NoError(err)
	require.NotNil(t, c)
	// Remove it
	err = RemoveContainer(c.ID, 0)
	assert.NoError(err)

	// Verify that we no longer find it.
	c, err = FindContainerByName(containerName)
	assert.NoError(err)
	assert.Nil(c)
}

func TestGetContainerEnv(t *testing.T) {
	assert := asrt.New(t)

	container, err := FindContainerByLabels(map[string]string{"com.ddev.site-name": testContainerName})
	assert.NoError(err)
	require.NotEmpty(t, container)

	env := GetContainerEnv("HOTDOG", *container)
	assert.Equal("superior-to-corndog", env)
	env = GetContainerEnv("POTATO", *container)
	assert.Equal("future-fry", env)
	env = GetContainerEnv("NONEXISTENT", *container)
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
	_, out, err := RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"//tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + "://tempmount"}, "25", true, false, nil)
	assert.NoError(err)
	assert.Contains(out, "simplescript.sh; TEMPENV=someenv UID=25")
	assert.Contains(out, "stdout is captured")
	assert.Contains(out, "stderr is also captured")

	// Try the case of running nonexistent script
	_, _, err = RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"nocommandbythatname"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + ":/tempmount"}, "25", true, false, nil)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "failed to StartContainer")
	}

	// Try the case of running a script that fails
	_, _, err = RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"/tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv", "ERROROUT=true"}, []string{testdata + ":/tempmount"}, "25", true, false, nil)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "container run failed with exit code 5")
	}

	// Provide an unqualified tag name
	_, _, err = RunSimpleContainer("busybox", "TestRunSimpleContainer"+basename, nil, nil, nil, nil, "", true, false, nil)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "image name must specify tag")
	}

	// Provide a malformed tag name
	_, _, err = RunSimpleContainer("busybox:", "TestRunSimpleContainer"+basename, nil, nil, nil, nil, "", true, false, nil)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "malformed tag provided")
	}
}

// TestGetExposedContainerPorts() checks to see that the ports expected
// to be exposed on a container actually are exposed.
func TestGetExposedContainerPorts(t *testing.T) {
	assert := asrt.New(t)

	testContainer, err := FindContainerByLabels(map[string]string{"com.ddev.site-name": testContainerName})
	require.NoError(t, err)
	require.NotNil(t, testContainer)
	ports, err := GetExposedContainerPorts(testContainer.ID)
	assert.NoError(err)
	assert.NotNil(ports)
	assert.Equal([]string{"8889", "8890"}, ports)
}

// TestDockerExec() checks docker.Exec()
func TestDockerExec(t *testing.T) {
	assert := asrt.New(t)
	client := GetDockerClient()

	id, _, err := RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"tail", "-f", "/dev/null"}, nil, nil, nil, "0", false, true, nil)
	assert.NoError(err)

	t.Cleanup(func() {
		err = client.RemoveContainer(docker.RemoveContainerOptions{
			ID:    id,
			Force: true,
		})
		assert.NoError(err)
	})

	stdout, _, err := Exec(id, "ls /etc", "")
	assert.NoError(err)
	assert.Contains(stdout, "group\nhostname")

	_, stderr, err := Exec(id, "ls /nothingthere", "")
	assert.Error(err)
	assert.Contains(stderr, "No such file or directory")

}

// TestCreateVolume does a trivial test of creating a trivial docker volume.
func TestCreateVolume(t *testing.T) {
	assert := asrt.New(t)
	// Make sure there's no existing volume.
	//nolint: errcheck
	RemoveVolume("junker99")
	volume, err := CreateVolume("junker99", "local", map[string]string{})
	require.NoError(t, err)

	//nolint: errcheck
	defer RemoveVolume("junker99")
	require.NotNil(t, volume)
	assert.Equal("junker99", volume.Name)
}

// TestRemoveVolume makes sure we can remove a volume successfully
func TestRemoveVolume(t *testing.T) {
	assert := asrt.New(t)
	client := GetDockerClient()

	testVolume := "junker999"
	spareVolume := "someVolumeThatCanNeverExit"

	_ = RemoveVolume(testVolume)
	pwd, err := os.Getwd()
	assert.NoError(err)

	source := pwd
	if runtime.GOOS == "darwin" && fileutil.IsDirectory(filepath.Join("/System/Volumes/Data", source)) {
		source = filepath.Join("/System/Volumes/Data", source)
	}
	volume, err := CreateVolume(
		testVolume,
		"local",
		map[string]string{"type": "nfs", "o": "addr=host.docker.internal,hard,nolock,rw",
			"device": ":" + MassageWindowsNFSMount(source)},
	)
	assert.NoError(err)

	volumes, err := client.ListVolumes(docker.ListVolumesOptions{Filters: map[string][]string{"name": {testVolume}}})
	assert.NoError(err)
	require.Len(t, volumes, 1)
	assert.Equal(testVolume, volumes[0].Name)

	require.NotNil(t, volume)
	assert.Equal(testVolume, volume.Name)
	err = RemoveVolume(testVolume)
	assert.NoError(err)

	volumes, err = client.ListVolumes(docker.ListVolumesOptions{Filters: map[string][]string{"name": {testVolume}}})
	assert.NoError(err)
	assert.Empty(volumes)

	// Make sure spareVolume doesn't exist, then make sure removal
	// of nonexistent volume doesn't result in error
	_ = RemoveVolume(spareVolume)
	err = RemoveVolume(spareVolume)
	assert.NoError(err)

}

// TestCopyIntoVolume makes sure CopyToVolume copies a local directory into a volume
func TestCopyIntoVolume(t *testing.T) {
	assert := asrt.New(t)
	err := RemoveVolume(t.Name())
	assert.NoError(err)

	pwd, _ := os.Getwd()
	err = CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name()), t.Name(), "", "0", "", true)
	assert.NoError(err)

	// Make sure that the content is the same, and that .test.sh is executable
	// On Windows the upload can result in losing executable bit
	_, out, err := RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"sh", "-c", "cd /mnt/" + t.Name() + " && ls -R .test.sh * && ./.test.sh"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil)
	assert.NoError(err)
	assert.Equal(`.test.sh
root.txt

subdir1:
subdir1.txt
hi this is a test file
`, out)

	err = CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name()), t.Name(), "somesubdir", "501", "", true)
	assert.NoError(err)
	_, out, err = RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"sh", "-c", "cd /mnt/" + t.Name() + "/somesubdir  && pwd && ls -R"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "0", true, false, nil)
	assert.NoError(err)
	assert.Equal(`/mnt/TestCopyIntoVolume/somesubdir
.:
root.txt
subdir1

./subdir1:
subdir1.txt
`, out)

	// Now try just a file
	err = CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name(), "root.txt"), t.Name(), "", "0", "", true)
	assert.NoError(err)

	// Make sure that the content is the same, and that .test.sh is executable
	_, out, err = RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"cat", "/mnt/" + t.Name() + "/root.txt"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil)
	assert.NoError(err)
	assert.Equal("root.txt here\n", out)

}

// TestDockerIP tries out a number of DOCKER_HOST permutations
// to verify that GetDockerIP does them right
func TestGetDockerIP(t *testing.T) {
	assert := asrt.New(t)

	origDOCKERHOST := os.Getenv("DOCKER_HOST")
	t.Cleanup(func() {
		_ = os.Setenv("DOCKER_HOST", origDOCKERHOST)
	})
	expectations := map[string]string{
		"":                            "127.0.0.1",
		"unix:///var/run/docker.sock": "127.0.0.1",
		"unix:///Users/rfay/.docker/run/docker.sock": "127.0.0.1",
		"unix:///Users/rfay/.colima/docker.sock":     "127.0.0.1",
		"tcp://185.199.110.153:2375":                 "185.199.110.153",
	}

	for k, v := range expectations {
		_ = os.Setenv("DOCKER_HOST", k)
		// DockerIP is cached, so we have to reset it to check
		DockerIP = ""
		result, err := GetDockerIP()
		assert.NoError(err)
		assert.Equal(v, result, "for %s expected %s, got %s", k, v, result)
	}
}

// TestCopyIntoContainer makes sure CopyIntoContainer copies a local file or directory into a specified
// path in container
func TestCopyIntoContainer(t *testing.T) {
	assert := asrt.New(t)
	pwd, _ := os.Getwd()

	cid, err := FindContainerByName(testContainerName)
	require.NoError(t, err)
	require.NotNil(t, cid)

	uid, _, _ := util.GetContainerUIDGid()
	targetDir, _, err := Exec(cid.ID, "mktemp -d", uid)
	require.NoError(t, err)
	targetDir = strings.Trim(targetDir, "\n")

	err = CopyIntoContainer(filepath.Join(pwd, "testdata", t.Name()), testContainerName, targetDir, "")
	require.NoError(t, err)

	out, _, err := Exec(cid.ID, fmt.Sprintf(`bash -c "cd %s && ls -R * .test.sh && ./.test.sh"`, targetDir), uid)
	require.NoError(t, err)
	assert.Equal(`.test.sh
root.txt

subdir1:
subdir1.txt
hi this is a test file
`, out)

	// Now try just a file
	err = CopyIntoContainer(filepath.Join(pwd, "testdata", t.Name(), "root.txt"), testContainerName, "/tmp", "")
	require.NoError(t, err)

	out, _, err = Exec(cid.ID, `cat /tmp/root.txt`, uid)
	require.NoError(t, err)
	assert.Equal("root.txt here\n", out)
}

// TestCopyFromContainer makes sure CopyFromContainer copies a container into a specified
// local directory
func TestCopyFromContainer(t *testing.T) {
	assert := asrt.New(t)
	containerSourceDir := "/var/tmp/backdrop_drush_commands/backdrop-drush-extension"
	containerExpectedFile := "backdrop.drush.inc"
	cid, err := FindContainerByName(testContainerName)
	require.NoError(t, err)
	require.NotNil(t, cid)

	targetDir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)

	err = CopyFromContainer(testContainerName, containerSourceDir, targetDir)
	require.NoError(t, err)

	assert.FileExists(filepath.Join(targetDir, path.Base(containerSourceDir), containerExpectedFile))
}
