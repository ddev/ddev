package globalconfig_test

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
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
	err = globalconfig.ReservePorts("TestGetFreePort", ports)
	assert.NoError(err)

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
