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

	dockerImages "github.com/ddev/ddev/pkg/docker"
	. "github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	docker "github.com/fsouza/go-dockerclient"
	logOutput "github.com/sirupsen/logrus"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testContainerName = "TestDockerUtils"

func init() {
	globalconfig.EnsureGlobalConfig()
	EnsureDdevNetwork()
}

func TestMain(m *testing.M) { os.Exit(testMain(m)) }

func testMain(m *testing.M) int {
	output.LogSetUp()

	_ = os.Setenv("DDEV_NONINTERACTIVE", "true")
	_ = os.Setenv("MUTAGEN_DATA_DIRECTORY", globalconfig.GetMutagenDataDirectory())

	labels := map[string]string{
		"com.ddev.site-name": testContainerName,
	}

	// prep docker container for docker util tests
	imageExists, err := ImageExistsLocally(dockerImages.GetWebImage())
	if err != nil {
		logOutput.Errorf("Failed to check for local image %s: %v", dockerImages.GetWebImage(), err)
		return 6
	}
	if !imageExists {
		err := Pull(dockerImages.GetWebImage())
		if err != nil {
			logOutput.Errorf("failed to pull test image: %v", err)
			return 7
		}
	}

	foundContainer, _ := FindContainerByLabels(labels)

	if foundContainer != nil {
		err = RemoveContainer(foundContainer.ID)
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
		foundContainer, _ := FindContainerByLabels(labels)
		if foundContainer != nil {
			err = RemoveContainer(foundContainer.ID)
			if err != nil {
				logOutput.Errorf("-- FAIL: dockerutils_test failed to remove test container: %v", err)
			}
		}
	}()

	log.Printf("ContainerWait at %v", time.Now())
	out, err := ContainerWait(60, map[string]string{"com.ddev.site-name": testContainerName})
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

// start container for tests; returns containerID, error
func startTestContainer() (string, error) {
	portBinding := map[docker.Port][]docker.PortBinding{
		"80/tcp": {
			{HostPort: "8889"},
		},
		"8025/tcp": {
			{HostPort: "8890"},
		}}
	containerID, _, err := RunSimpleContainer(dockerImages.GetWebImage(), testContainerName, nil, nil, []string{
		"HOTDOG=superior-to-corndog",
		"POTATO=future-fry",
		"DDEV_WEBSERVER_TYPE=nginx-fpm",
	}, nil, "33", false, true, map[string]string{"com.docker.compose.service": "web", "com.ddev.site-name": testContainerName}, portBinding)
	if err != nil {
		return "", err
	}

	return containerID, nil
}

// TestGetContainerHealth tests the function for processing container readiness.
func TestGetContainerHealth(t *testing.T) {
	assert := asrt.New(t)
	client := GetDockerClient()

	labels := map[string]string{
		"com.ddev.site-name": testContainerName,
	}

	t.Cleanup(func() {
		container, err := FindContainerByLabels(labels)
		if err == nil || container != nil {
			err = RemoveContainer(container.ID)
			assert.NoError(err)
		}

		// Make sure test container exists again
		_, err = startTestContainer()
		assert.NoError(err)
		healthDetail, err := ContainerWait(30, labels)
		assert.NoError(err, "healtdetail='%s'", healthDetail)

		container, err = FindContainerByLabels(labels)
		assert.NoError(err)
		assert.NotNil(container)

		status, healthDetail := GetContainerHealth(container)
		assert.Contains(healthDetail, "/var/www/html:OK mailhog:OK phpstatus:OK ")
		assert.Equal("healthy", status)
	})

	container, err := FindContainerByLabels(labels)
	require.NoError(t, err)
	require.NotNil(t, container)

	status, log := GetContainerHealth(container)
	assert.Equal(status, "healthy", "container should be healthy; log=%v", log)

	// Now break the container and make sure it's unhealthy
	err = client.StopContainer(container.ID, 10)
	assert.NoError(err)

	status, log = GetContainerHealth(container)
	assert.Equal(status, "unhealthy", "container should be unhealthy; log=%v", log)
	assert.NoError(err)

}

// TestContainerWait tests the error cases for the container check wait loop.
func TestContainerWait(t *testing.T) {
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

	// Try 30-second wait for "healthy", should show OK
	healthDetail, err := ContainerWait(30, labels)
	assert.NoError(err)

	assert.Contains(healthDetail, "phpstatus:OK")

	// Try a nonexistent container, should get error
	labels = map[string]string{"com.ddev.site-name": "nothing-there"}
	_, err = ContainerWait(1, labels)
	require.Error(t, err)
	assert.Contains(err.Error(), "failed to query container")

	// If we just run a quick container and it immediately exits, ContainerWait should find it not there
	// and note that it exited.
	labels = map[string]string{"test": "quickexit"}
	_ = RemoveContainersByLabels(labels)
	cID, _, err := RunSimpleContainer(versionconstants.BusyboxImage, t.Name()+util.RandString(5), []string{"ls"}, nil, nil, nil, "0", false, true, labels, nil)
	t.Cleanup(func() {
		_ = RemoveContainer(cID)
	})
	require.NoError(t, err)
	_, err = ContainerWait(5, labels)
	assert.Error(err)
	assert.Contains(err.Error(), "container exited")
	_ = RemoveContainer(cID)

	// If we run a container that does not have a healthcheck
	// it should be found as good immediately
	labels = map[string]string{"test": "nohealthcheck"}
	_ = RemoveContainersByLabels(labels)
	cID, _, err = RunSimpleContainer(versionconstants.BusyboxImage, t.Name()+util.RandString(5), []string{"sleep", "60"}, nil, nil, nil, "0", false, true, labels, nil)
	t.Cleanup(func() {
		_ = RemoveContainer(cID)
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
	cID, _, err = RunSimpleContainer(ddevWebserver, t.Name()+util.RandString(5), []string{"sleep", "5"}, nil, []string{"DDEV_WEBSERVER_TYPE=nginx-fpm"}, nil, "0", false, true, labels, nil)
	t.Cleanup(func() {
		_ = RemoveContainer(cID)
	})
	require.NoError(t, err)
	_, err = ContainerWait(3, labels)
	assert.Error(err)
	assert.Contains(err.Error(), "timed out without becoming healthy")
	_ = RemoveContainer(cID)

	// If we run a container that *does* have a healthcheck but it's not healthy for a while
	// then ContainerWait should detect failure early, but should succeed later
	labels = map[string]string{"test": "hashealthcheckbutbad"}
	_ = RemoveContainersByLabels(labels)
	cID, _, err = RunSimpleContainer(ddevWebserver, t.Name()+util.RandString(5), []string{"bash", "-c", "sleep 5 && /start.sh"}, nil, []string{"DDEV_WEBSERVER_TYPE=nginx-fpm"}, nil, "0", false, true, labels, nil)
	t.Cleanup(func() {
		_ = RemoveContainer(cID)
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
		_ = RemoveContainer(container.ID)
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

	_, err = ContainerWait(60, map[string]string{"com.ddev.site-name": t.Name()})
	if err != nil {
		logout, _ := exec.RunCommand("docker", []string{"logs", t.Name()})
		inspectOut, _ := exec.RunCommandPipe("sh", []string{"-c", fmt.Sprintf("docker inspect %s|jq -r '.[0].State.Health.Log'", t.Name())})
		t.Fatalf("FAIL: TestComposeWithStreams failed to ContainerWait for container: %v, logs\n========= container logs ======\n%s\n======= end logs =======\n==== health log =====\ninspectOut\n%s\n========", err, logout, inspectOut)
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

	globalconfig.DockerComposeVersion = ""
	composeErr := CheckDockerCompose()
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
		err = RemoveContainer(c.ID)
		assert.NoError(err)
	}

	// Run a container, don't remove it.
	cID, _, err := RunSimpleContainer(versionconstants.BusyboxImage, containerName, []string{"//tempmount/sleepALittle.sh"}, nil, nil, []string{testdata + "://tempmount"}, "25", false, false, nil, nil)
	assert.NoError(err)

	defer func() {
		_ = RemoveContainer(cID)
	}()

	// Now find the container by name
	c, err = FindContainerByName(containerName)
	assert.NoError(err)
	require.NotNil(t, c)
	// Remove it
	err = RemoveContainer(c.ID)
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
	_, out, err := RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"//tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + "://tempmount"}, "25", true, false, nil, nil)
	assert.NoError(err)
	assert.Contains(out, "simplescript.sh; TEMPENV=someenv UID=25")
	assert.Contains(out, "stdout is captured")
	assert.Contains(out, "stderr is also captured")

	// Try the case of running nonexistent script
	_, _, err = RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"nocommandbythatname"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + ":/tempmount"}, "25", true, false, nil, nil)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "failed to StartContainer")
	}

	// Try the case of running a script that fails
	_, _, err = RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"/tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv", "ERROROUT=true"}, []string{testdata + ":/tempmount"}, "25", true, false, nil, nil)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "container run failed with exit code 5")
	}

	// Provide an unqualified tag name
	_, _, err = RunSimpleContainer("busybox", "TestRunSimpleContainer"+basename, nil, nil, nil, nil, "", true, false, nil, nil)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "image name must specify tag")
	}

	// Provide a malformed tag name
	_, _, err = RunSimpleContainer("busybox:", "TestRunSimpleContainer"+basename, nil, nil, nil, nil, "", true, false, nil, nil)
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

	id, _, err := RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"tail", "-f", "/dev/null"}, nil, nil, nil, "0", false, true, nil, nil)
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
	volume, err := CreateVolume("junker99", "local", map[string]string{}, nil)
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
	nfsServerAddr, _ := GetNFSServerAddr()
	volume, err := CreateVolume(testVolume, "local", map[string]string{"type": "nfs", "o": fmt.Sprintf("addr=%s,hard,nolock,rw,wsize=32768,rsize=32768", nfsServerAddr),
		"device": ":" + MassageWindowsNFSMount(source)}, nil)
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
	_, out, err := RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"sh", "-c", "cd /mnt/" + t.Name() + " && ls -R .test.sh * && ./.test.sh"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil, nil)
	assert.NoError(err)
	assert.Equal(`.test.sh
root.txt

subdir1:
subdir1.txt
hi this is a test file
`, out)

	err = CopyIntoVolume(filepath.Join(pwd, "testdata", t.Name()), t.Name(), "somesubdir", "501", "", true)
	assert.NoError(err)
	_, out, err = RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"sh", "-c", "cd /mnt/" + t.Name() + "/somesubdir  && pwd && ls -R"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "0", true, false, nil, nil)
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
	_, out, err = RunSimpleContainer(versionconstants.BusyboxImage, "", []string{"cat", "/mnt/" + t.Name() + "/root.txt"}, nil, nil, []string{t.Name() + ":/mnt/" + t.Name()}, "25", true, false, nil, nil)
	assert.NoError(err)
	assert.Equal("root.txt here\n", out)

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
