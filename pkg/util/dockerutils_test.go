package util_test

import (
	"testing"

	"path/filepath"

	"github.com/drud/ddev/pkg/testcommon"
	. "github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

// TestGetContainerHealth tests the function for processing container readiness.
func TestGetContainerHealth(t *testing.T) {
	assert := assert.New(t)
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
	assert := assert.New(t)

	labels := map[string]string{
		"com.ddev.site-name":         "foo",
		"com.docker.compose.service": "web",
	}

	err := ContainerWait(0, labels)
	assert.Error(err)
	assert.Equal("health check timed out", err.Error())

	err = ContainerWait(5, labels)
	assert.Error(err)
	assert.Equal("failed to query container", err.Error())
}

// TestComposeCmd tests execution of docker-compose commands.
func TestComposeCmd(t *testing.T) {
	assert := assert.New(t)

	composeFiles := []string{filepath.Join("testing", "docker-compose.yml")}

	stdout := testcommon.CaptureStdOut()
	err := ComposeCmd(composeFiles, "config", "--services")
	assert.NoError(err)
	out := stdout()
	assert.Contains(out, "web")
	assert.Contains(out, "db")

	composeFiles = append(composeFiles, filepath.Join("testing", "docker-compose.override.yml"))

	stdout = testcommon.CaptureStdOut()
	err = ComposeCmd(composeFiles, "config", "--services")
	assert.NoError(err)
	out = stdout()
	assert.Contains(out, "web")
	assert.Contains(out, "db")
	assert.Contains(out, "foo")

	composeFiles = []string{"invalid.yml"}
	err = ComposeCmd(composeFiles, "config", "--services")
	assert.Error(err)
}

func TestGetAppContainers(t *testing.T) {
	assert := assert.New(t)
	sites, err := GetAppContainers("dockertest")
	assert.NoError(err)
	assert.Equal(sites[0].Image, TestRouterImage+":"+TestRouterTag)
}

func TestGetContainerEnv(t *testing.T) {
	assert := assert.New(t)

	container, err := FindContainerByLabels(map[string]string{"com.docker.compose.service": "ddevrouter"})
	assert.NoError(err)

	env := GetContainerEnv("HOTDOG", container)
	assert.Equal("superior-to-corndog", env)
	env = GetContainerEnv("POTATO", container)
	assert.Equal("future-fry", env)
	env = GetContainerEnv("NONEXISTENT", container)
	assert.Equal("", env)
}

func TestCheckDockerVersion(t *testing.T) {
	assert := assert.New(t)

	err := CheckDockerVersion(">= 50.0.0")
	assert.Error(err)
	assert.Contains(err.Error(), "is less than 50.0.0")

	err = CheckDockerVersion(version.DockerVersionConstraint)
	assert.NoError(err)
}
