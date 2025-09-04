package dockerutil_test

import (
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	asrt "github.com/stretchr/testify/assert"
)

// TestDockerIP tries out a number of DockerHost permutations
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

	// Save original DockerHost to restore it later
	_, origDockerHost, _ := dockerutil.GetDockerContextNameAndHost()
	t.Cleanup(func() {
		dockerutil.ResetDockerIPForDockerHost(origDockerHost)
	})

	for k, v := range expectations {
		// DockerIP is cached, so we have to reset it to check
		dockerutil.ResetDockerIPForDockerHost(k)
		result, err := dockerutil.GetDockerIP()
		assert.NoError(err)
		assert.Equal(v, result, "for %s expected %s, got %s", k, v, result)
	}
}
