package dockerutil_test

import (
	"github.com/drud/ddev/pkg/util"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"

	logOutput "github.com/sirupsen/logrus"

	"path/filepath"

	. "github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/version"
	"github.com/fsouza/go-dockerclient"
	asrt "github.com/stretchr/testify/assert"
)

var TestContainerName = "dockerutils-test"

func TestMain(m *testing.M) {
	output.LogSetUp()

	// prep docker container for docker util tests
	client := GetDockerClient()
	imageExists, err := ImageExistsLocally(version.WebImg + ":" + version.WebTag)
	if err != nil {
		logOutput.Fatalf("Failed to check for local image %s: %v", version.WebImg+":"+version.WebTag, err)
	}
	if !imageExists {
		err := client.PullImage(docker.PullImageOptions{
			Repository: version.WebImg,
			Tag:        version.WebTag,
		}, docker.AuthConfiguration{})
		if err != nil {
			logOutput.Fatal("failed to pull test image ", err)
		}
	}

	foundContainer, _ := FindContainerByLabels(map[string]string{"com.ddev.site-name": "dockerutils-test"})

	if foundContainer != nil {
		_ = client.StopContainer(foundContainer.ID, 10)

		err = client.RemoveContainer(docker.RemoveContainerOptions{ID: foundContainer.ID})
		if err != nil {
			logOutput.Fatalf("Failed to remove container %s: %v", foundContainer.ID, err)
		}
	}

	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Name: TestContainerName,
		Config: &docker.Config{
			Image: version.WebImg + ":" + version.WebTag,
			Labels: map[string]string{
				"com.docker.compose.service": "ddevrouter",
				"com.ddev.site-name":         "dockerutils-test",
			},
			Env: []string{"HOTDOG=superior-to-corndog", "POTATO=future-fry"},
		},
		//          "PortBindings": { "22/tcp": [{ "HostPort": "11022" }] },
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
		logOutput.Fatal("failed to create/start docker container ", err)
	}
	err = client.StartContainer(container.ID, nil)
	if err != nil {
		logOutput.Fatalf("failed to StartContainer: %v", err)
	}
	exitStatus := m.Run()
	// teardown docker container from docker util tests
	err = client.StopContainer(container.ID, 10)
	if err != nil {
		logOutput.Fatalf("Failed to stop container: %v", err)
	}
	err = client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    container.ID,
		Force: true,
	})
	if err != nil {
		logOutput.Fatalf("failed to remove test container: %v", err)
	}

	os.Exit(exitStatus)
}

// TestGetContainerHealth tests the function for processing container readiness.
func TestGetContainerHealth(t *testing.T) {
	assert := asrt.New(t)
	client := GetDockerClient()

	labels := map[string]string{
		"com.ddev.site-name": "dockerutils-test",
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
	healthDetail, err := ContainerWait(15, labels)
	assert.NoError(err)

	assert.Equal("phpstatus: OK, /var/www/html: OK, mailhog: OK", healthDetail)

	status, healthDetail = GetContainerHealth(container)
	assert.Equal(status, "healthy")
	assert.Equal("phpstatus: OK, /var/www/html: OK, mailhog: OK", healthDetail)
}

// TestContainerWait tests the error cases for the container check wait loop.
func TestContainerWait(t *testing.T) {
	assert := asrt.New(t)

	labels := map[string]string{
		"com.ddev.site-name": "dockerutils-test",
	}

	// Try a zero-wait, should show timed-out
	_, err := ContainerWait(0, labels)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "health check timed out")
	}

	// Try 15-second wait for "healthy", should show OK
	healthDetail, err := ContainerWait(15, labels)
	assert.NoError(err)

	if !util.IsDockerToolbox() {
		assert.Contains(healthDetail, "phpstatus: OK")
	}

	// Try a nonexistent container, should get error
	labels = map[string]string{
		"com.ddev.site-name": "nothing-there",
	}
	_, err = ContainerWait(1, labels)
	require.Error(t, err)
	assert.Contains(err.Error(), "failed to query container")
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

	// Use the current actual web container for this, so replace in base docker-compose file
	composeBase := filepath.Join("testdata", "test-compose-with-streams.yaml")
	tmp, err := ioutil.TempDir("", "")
	assert.NoError(err)
	realComposeFile := filepath.Join(tmp, "replaced-compose-with-streams.yaml")

	err = fileutil.ReplaceStringInFile("TEST-COMPOSE-WITH-STREAMS-IMAGE", version.WebImg+":"+version.WebTag, composeBase, realComposeFile)
	assert.NoError(err)
	defer os.Remove(realComposeFile)

	composeFiles := []string{realComposeFile}

	_, _, err = ComposeCmd(composeFiles, "up", "-d")
	require.NoError(t, err)
	//nolint: errcheck
	defer ComposeCmd(composeFiles, "down")

	_, err = ContainerWait(10, map[string]string{"com.ddev.site-name": "test-compose-with-streams"})
	assert.NoError(err)

	// Point stdout to os.Stdout and do simple ps -ef in web container
	stdout := util.CaptureStdOut()
	err = ComposeWithStreams(composeFiles, os.Stdin, os.Stdout, os.Stderr, "exec", "-T", "web", "ps", "-ef")
	assert.NoError(err)
	output := stdout()
	assert.Contains(output, "supervisord")

	// Reverse stdout and stderr and create an error and normal stdout. We should see only the error captured in stdout
	stdout = util.CaptureStdOut()
	err = ComposeWithStreams(composeFiles, os.Stdin, os.Stderr, os.Stdout, "exec", "-T", "web", "ls", "-d", "xx", "/var/run/apache2")
	assert.NoError(err)
	output = stdout()
	assert.Equal(output, "ls: cannot access 'xx': No such file or directory\n")

	// Flip stdout and stderr and create an error and normal stdout. We should see only the success captured in stdout
	stdout = util.CaptureStdOut()
	err = ComposeWithStreams(composeFiles, os.Stdin, os.Stdout, os.Stderr, "exec", "-T", "web", "ls", "-d", "xx", "/var/run/apache2")
	assert.NoError(err)
	output = stdout()
	assert.Equal(output, "/var/run/apache2\n")

}

// TestCheckCompose tests detection of docker-compose.
func TestCheckCompose(t *testing.T) {
	assert := asrt.New(t)

	err := CheckDockerCompose(version.DockerComposeVersionConstraint)
	assert.NoError(err)
}

// TestGetAppContainers looks for container with sitename dockerutils-test
func TestGetAppContainers(t *testing.T) {
	assert := asrt.New(t)
	containers, err := GetAppContainers("dockerutils-test")
	assert.NoError(err)
	assert.Contains(containers[0].Image, version.WebImg)
}

func TestGetContainerEnv(t *testing.T) {
	assert := asrt.New(t)

	container, err := FindContainerByLabels(map[string]string{"com.docker.compose.service": "ddevrouter"})
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

	// Try the success case; script found, runs, all good.
	_, out, err := RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"/tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + ":/tempmount"}, "25", true)
	assert.NoError(err)
	assert.Contains(out, "simplescript.sh; TEMPENV=someenv UID=25")

	// Try the case of running nonexistent script
	_, _, err = RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"nocommandbythatname"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + ":/tempmount"}, "25", true)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "failed to StartContainer")
	}

	// Try the case of running a script that fails
	_, _, err = RunSimpleContainer("busybox:latest", "TestRunSimpleContainer"+basename, []string{"/tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv", "ERROROUT=true"}, []string{testdata + ":/tempmount"}, "25", true)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "container run failed with exit code 5")
	}

	// Provide an unqualified tag name
	_, _, err = RunSimpleContainer("busybox", "TestRunSimpleContainer"+basename, nil, nil, nil, nil, "", true)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "image name must specify tag")
	}

	// Provide a malformed tag name
	_, _, err = RunSimpleContainer("busybox:", "TestRunSimpleContainer"+basename, nil, nil, nil, nil, "", true)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "malformed tag provided")
	}
}

// TestGetExposedContainerPorts() checks to see that the ports expected
// to be exposed on a container actually are exposed.
func TestGetExposedContainerPorts(t *testing.T) {
	assert := asrt.New(t)

	testContainer, err := FindContainerByLabels(map[string]string{"com.ddev.site-name": "dockerutils-test"})
	require.NoError(t, err)
	require.NotNil(t, testContainer)
	ports, err := GetExposedContainerPorts(testContainer.ID)
	assert.NoError(err)
	assert.NotNil(ports)
	assert.Equal([]string{"8889", "8890"}, ports)
}
