package ddevapp

import (
	"net"
	"net/netip"
	"strconv"
	"testing"
	"time"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/netutil"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"
)

func TestHasUnspecifiedHostPort(t *testing.T) {
	require.True(t, hasUnspecifiedHostPort("0"))
	require.True(t, hasUnspecifiedHostPort(""))
	require.False(t, hasUnspecifiedHostPort("12345"))
}

// TestFixupComposeYamlAssignsUnspecifiedHostPorts checks that both ways of
// leaving the host port empty end up with a port from DDEV's range.
func TestFixupComposeYamlAssignsUnspecifiedHostPorts(t *testing.T) {
	if !dockerutil.CanCheckHostPortsLocally() {
		t.Skip("DDEV leaves host port choice to the engine here")
	}
	app, err := NewApp(t.TempDir(), true)
	require.NoError(t, err)

	project := &composeTypes.Project{
		Networks: composeTypes.Networks{},
		Services: composeTypes.Services{
			"hostport-test": {
				Name:        "hostport-test",
				Networks:    map[string]*composeTypes.ServiceNetworkConfig{},
				Environment: composeTypes.MappingWithEquals{},
				Ports: []composeTypes.ServicePortConfig{
					{Target: 80, Protocol: "tcp"},
					{Target: 81, Protocol: "tcp", Published: "0"},
					{Target: 82, Protocol: "tcp", Published: "23456"},
				},
			},
		},
	}
	project, err = fixupComposeYaml(project, app)
	require.NoError(t, err)

	ports := project.Services["hostport-test"].Ports
	for _, p := range ports[:2] {
		published, err := strconv.Atoi(p.Published)
		require.NoError(t, err, "target %d should have a numeric host port, got %q", p.Target, p.Published)
		require.GreaterOrEqual(t, published, MinHostPort)
		require.LessOrEqual(t, published, MaxHostPort)
	}
	require.NotEqual(t, ports[0].Published, ports[1].Published)
	require.Equal(t, "23456", ports[2].Published)
}

// TestExistingContainerHostPortStopped checks that a stopped container's stored
// host port is reused while free and dropped once something else holds it.
func TestExistingContainerHostPortStopped(t *testing.T) {
	app, err := NewApp(t.TempDir(), true)
	require.NoError(t, err)
	a := newHostPortAllocator(app)
	if !a.localPorts {
		t.Skip("DDEV leaves host port choice to the engine here")
	}

	hostIP := "127.0.0.1"
	port := netutil.AllocateHostPort(hostIP, MinHostPort, MaxHostPort, nil)
	require.NotEmpty(t, port)
	labels := map[string]string{
		"com.ddev.site-name":         app.GetName(),
		"com.docker.compose.service": "web",
		"com.docker.compose.oneoff":  "False",
	}
	portBindings := network.PortMap{
		network.MustParsePort("80/tcp"): {{HostIP: netip.MustParseAddr(hostIP), HostPort: port}},
	}
	containerID, _, err := dockerutil.RunSimpleContainer(versionconstants.UtilitiesImage, "TestExistingContainerHostPortStopped-"+app.GetName(), []string{"true"}, nil, nil, nil, "", false, false, labels, portBindings, &dockerutil.NoHealthCheck)
	t.Cleanup(func() {
		_ = dockerutil.RemoveContainer(containerID)
	})
	require.NoError(t, err)
	// Some providers, such as OrbStack, release the host port a few hundred
	// milliseconds after the container exits.
	portNum, err := strconv.Atoi(port)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return netutil.IsHostPortFree(hostIP, portNum)
	}, 5*time.Second, 100*time.Millisecond, "host port %s still bound after the container exited", port)

	p := composeTypes.ServicePortConfig{Target: 80, Protocol: "tcp", HostIP: hostIP}
	require.Equal(t, port, a.existingContainerHostPort("web", p))

	l, err := net.Listen("tcp", net.JoinHostPort(hostIP, port))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = l.Close()
	})
	require.Empty(t, a.existingContainerHostPort("web", p))
}
