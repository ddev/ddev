package ddevapp

import (
	"strconv"
	"testing"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/dockerutil"
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
	if dockerutil.IsRemoteDockerHost() {
		t.Skip("DDEV leaves host port choice to the engine on a remote Docker host")
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
