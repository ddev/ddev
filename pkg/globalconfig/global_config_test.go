package globalconfig_test

import (
	"context"
	"errors"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"os"
	"strconv"
	"testing"
	"time"
)

// TestGetFreePort checks GetFreePort() to make sure it respects
// ports reserved in DdevGlobalConfig.UsedHostPorts
// and that the port can actually be bound.
func TestGetFreePort(t *testing.T) {
	assert := asrt.New(t)

	dockerIP, err := dockerutil.GetDockerIP()
	require.NoError(t, err)

	// Find out a starting port the OS is likely to give us.
	startPort, err := globalconfig.GetFreePort(dockerIP)
	require.NoError(t, err)

	// Put 100 used ports in the UsedHostPorts
	i, err := strconv.Atoi(startPort)
	i = i + 1
	max := i + 100
	require.NoError(t, err)
	ports := []string{}
	for ; i < max; i++ {
		ports = append(ports, strconv.Itoa(i))
	}
	err = globalconfig.ReservePorts(t.Name(), ports)
	assert.NoError(err)
	t.Cleanup(func() {
		_ = globalconfig.RemoveProjectInfo(t.Name())
	})

	for try := 0; try < 5; try++ {
		port, err := globalconfig.GetFreePort(dockerIP)
		require.NoError(t, err)
		assert.NotContains(globalconfig.DdevGlobalConfig.ProjectList["TestGetFreePort"].UsedHostPorts, port)

		// Make sure we can actually use the port.
		dockerCommand := []string{"run", "--rm", "-p" + dockerIP + ":" + port + ":" + port, "busybox:latest"}
		_, err = exec.RunCommand("docker", dockerCommand)

		assert.NoError(err, "failed to 'docker %v': %v", dockerCommand, err)
	}
}

// TestSetProjectAppRoot tests behavior of SetProjectAppRoot
// This also tests RemoveProject
func TestSetProjectAppRoot(t *testing.T) {
	assert := asrt.New(t)

	// Make sure conflicting approot results in error
	// Make sure empty project works
	// Make sure existing project with no approot works

	// Non-existing approot should cause a fail
	err := globalconfig.SetProjectAppRoot(t.Name(), "/nowhere/junk-approot-1")
	assert.Error(err)
	_ = globalconfig.RemoveProjectInfo(t.Name())

	// Create a project in a valid directory
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err = globalconfig.SetProjectAppRoot(t.Name(), tmpDir)
	assert.NoError(err)

	t.Cleanup(func() {
		_ = globalconfig.RemoveProjectInfo(t.Name())
		_ = os.RemoveAll(tmpDir)
	})

	project := globalconfig.GetProject(t.Name())
	require.NotNil(t, project)

	// Try to set approot to existing but conflicting approot
	tmpDir2 := testcommon.CreateTmpDir(t.Name())
	// nolint: errcheck
	defer os.RemoveAll(tmpDir2)
	err = globalconfig.SetProjectAppRoot(t.Name(), tmpDir2)
	assert.Error(err)

	// Make sure that the approot didn't accidentally get changed to
	// bad approot
	p2 := globalconfig.GetProject(t.Name())
	assert.Equal(tmpDir, p2.AppRoot)

	err = globalconfig.RemoveProjectInfo(t.Name())
	assert.NoError(err)

	// Make sure after removal the project is gone
	p3 := globalconfig.GetProject(t.Name())
	assert.Nil(p3)

	// ReservePorts will create the project, but without an approot
	err = globalconfig.ReservePorts(t.Name(), []string{})
	assert.NoError(err)
	project = globalconfig.GetProject(t.Name())
	require.NotNil(t, project)
	assert.Empty(project.AppRoot)

	err = globalconfig.SetProjectAppRoot(t.Name(), tmpDir)
	assert.NoError(err)

	project = globalconfig.GetProject(t.Name())
	assert.Equal(tmpDir, project.AppRoot)
}

type internetActiveNetResolverStub struct {
	sleepTime time.Duration
	err       error
}

// LookupHost is a custom version of net.LookupHost that just wastes some time and then returns
func (t internetActiveNetResolverStub) LookupHost(ctx context.Context, _ string) ([]string, error) {
	select {
	case <-time.After(t.sleepTime):
	case <-ctx.Done():
		return nil, errors.New("context timed out")
	}
	return nil, t.err
}

// internetActiveResetVariables resets the global variables IsInternetActive() uses back to their defaults
func internetActiveResetVariables() {
	globalconfig.IsInternetActiveNetResolver = net.DefaultResolver
	globalconfig.IsInternetActiveAlreadyChecked = false
	globalconfig.IsInternetActiveResult = false
	globalconfig.DdevGlobalConfig.InternetDetectionTimeout = nodeps.InternetDetectionTimeoutDefault
}

// TestIsInternetActiveErrorOccurred tests if IsInternetActive() returns false when LookupHost returns an error
func TestIsInternetActiveErrorOccurred(t *testing.T) {
	internetActiveResetVariables()

	globalconfig.IsInternetActiveNetResolver = internetActiveNetResolverStub{
		sleepTime: 0,
		err:       errors.New("test error"),
	}

	asrt.False(t, globalconfig.IsInternetActive())
}

// TestIsInternetActiveTimeout tests if IsInternetActive() returns false when it times out
func TestIsInternetActiveTimeout(t *testing.T) {
	internetActiveResetVariables()

	globalconfig.IsInternetActiveNetResolver = internetActiveNetResolverStub{
		sleepTime: 1000 * time.Millisecond,
	}

	asrt.False(t, globalconfig.IsInternetActive())
}

// TestIsInternetActiveAlreadyChecked tests if IsInternetActive() returns true when it has already
// been called and returned true on an earlier execution.
func TestIsInternetActiveAlreadyChecked(t *testing.T) {
	internetActiveResetVariables()

	globalconfig.IsInternetActiveAlreadyChecked = true
	globalconfig.IsInternetActiveResult = true

	asrt.True(t, globalconfig.IsInternetActive())
}

// TestIsInternetActive tests if IsInternetActive() returns true, when the LookupHost call goes well
// and if it properly sets the globals so it won't execute the LookupHost again.
func TestIsInternetActive(t *testing.T) {
	internetActiveResetVariables()

	globalconfig.IsInternetActiveNetResolver = internetActiveNetResolverStub{
		sleepTime: 0,
	}

	// should return true
	asrt.True(t, globalconfig.IsInternetActive())
	// should have set the IsInternetActiveAlreadyChecked to true
	asrt.True(t, globalconfig.IsInternetActiveAlreadyChecked)
	// result should still be true
	asrt.True(t, globalconfig.IsInternetActiveResult)
	// and calling it again, should also still be true
	asrt.True(t, globalconfig.IsInternetActive())
}
