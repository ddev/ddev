package dockerutil_test

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"

	"path/filepath"

	. "github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/version"
	"github.com/fsouza/go-dockerclient"
	asrt "github.com/stretchr/testify/assert"
)

var (
	// The image here can be any image, it just has to exist so it can be used for labels, etc.
	TestRouterImage = "busybox"
	TestRouterTag   = "1"
)

func TestMain(m *testing.M) {
	output.LogSetUp()

	// prep docker container for docker util tests
	client := GetDockerClient()

	err := client.PullImage(docker.PullImageOptions{
		Repository: TestRouterImage,
		Tag:        TestRouterTag,
	}, docker.AuthConfiguration{})
	if err != nil {
		log.Fatal("failed to pull test image ", err)
	}

	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Name: "envtest",
		Config: &docker.Config{
			Image: TestRouterImage + ":" + TestRouterTag,
			Labels: map[string]string{
				"com.docker.compose.service": "ddevrouter",
				"com.ddev.site-name":         "dockertest",
			},
			Env: []string{"HOTDOG=superior-to-corndog", "POTATO=future-fry"},
		},
	})
	if err != nil {
		log.Fatal("failed to create/start docker container ", err)
	}
	exitStatus := m.Run()
	// teardown docker container from docker util tests
	err = client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    container.ID,
		Force: true,
	})
	if err != nil {
		log.Fatal("failed to remove test container: ", err)
	}

	os.Exit(exitStatus)
}

// TestGetContainerHealth tests the function for processing container readiness.
func TestGetContainerHealth(t *testing.T) {
	assert := asrt.New(t)
	container := docker.APIContainers{
		Status: "Up 24 seconds (health: starting)",
	}
	out := GetContainerHealth(container)
	assert.Equal(out, "starting")

	container = docker.APIContainers{
		Status: "Up 14 minutes (healthy)",
	}
	out = GetContainerHealth(container)
	assert.Equal(out, "healthy")

	container = docker.APIContainers{
		State: "exited",
	}
	out = GetContainerHealth(container)
	assert.Equal(out, container.State)

	container = docker.APIContainers{
		State: "restarting",
	}
	out = GetContainerHealth(container)
	assert.Equal(out, container.State)
}

// TestContainerWait tests the error cases for the container check wait loop.
func TestContainerWait(t *testing.T) {
	assert := asrt.New(t)

	labels := map[string]string{
		"com.ddev.site-name":         "foo",
		"com.docker.compose.service": "web",
	}

	err := ContainerWait(0, labels)
	assert.Error(err)
	assert.Contains(err.Error(), "health check timed out")

	err = ContainerWait(5, labels)
	assert.Error(err)
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

// TestCheckCompose tests detection of docker-compose.
func TestCheckCompose(t *testing.T) {
	assert := asrt.New(t)

	err := CheckDockerCompose(version.DockerComposeVersionConstraint)
	assert.NoError(err)
}

func TestGetAppContainers(t *testing.T) {
	assert := asrt.New(t)
	sites, err := GetAppContainers("dockertest")
	assert.NoError(err)
	assert.Equal(sites[0].Image, TestRouterImage+":"+TestRouterTag)
}

func TestGetContainerEnv(t *testing.T) {
	assert := asrt.New(t)

	container, err := FindContainerByLabels(map[string]string{"com.docker.compose.service": "ddevrouter"})
	assert.NoError(err)

	env := GetContainerEnv("HOTDOG", container)
	assert.Equal("superior-to-corndog", env)
	env = GetContainerEnv("POTATO", container)
	assert.Equal("future-fry", env)
	env = GetContainerEnv("NONEXISTENT", container)
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
	out, err := RunSimpleContainer("busybox", "TestRunSimpleContainer"+basename, []string{"/tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + ":/tempmount"}, "25")
	assert.NoError(err)
	assert.Contains(out, "simplescript.sh; TEMPENV=someenv UID=25")

	// Try the case of running nonexistent script
	out, err = RunSimpleContainer("busybox", "TestRunSimpleContainer"+basename, []string{"nocommandbythatname"}, nil, []string{"TEMPENV=someenv"}, []string{testdata + ":/tempmount"}, "25")
	assert.Error(err)
	assert.Contains(err.Error(), "failed to StartContainer")

	// Try the case of running a script that fails
	out, err = RunSimpleContainer("busybox", "TestRunSimpleContainer"+basename, []string{"/tempmount/simplescript.sh"}, nil, []string{"TEMPENV=someenv", "ERROROUT=true"}, []string{testdata + ":/tempmount"}, "25")
	assert.Error(err)
	assert.Contains(err.Error(), "container run failed with exit code 5")
}
