package cmd

import (
	"fmt"
	"testing"

	"github.com/drud/ddev/pkg/local"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/network"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

// TestDevStart runs drud dev restart on the test apps
func TestDevRestart(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)
	args := []string{"restart", DevTestApp, DevTestEnv}
	out, err := system.RunCommand(DdevBin, args)
	assert.NoError(err)
	format := fmt.Sprintf
	assert.Contains(string(out), format("Stopping legacy-%s-%s-web ... done", DevTestApp, DevTestEnv))
	assert.Contains(string(out), format("Stopping legacy-%s-%s-db ... done", DevTestApp, DevTestEnv))
	assert.Contains(string(out), format("Starting legacy-%s-%s-web", DevTestApp, DevTestEnv))
	assert.Contains(string(out), format("Starting legacy-%s-%s-db", DevTestApp, DevTestEnv))
	assert.Contains(string(out), "Your application can be reached at")

	app := local.LegacyApp{}
	app.AppBase.Name = DevTestApp
	app.AppBase.Environment = DevTestEnv

	assert.Equal(true, dockerutil.IsRunning(app.ContainerName()+"-web"))
	assert.Equal(true, dockerutil.IsRunning(app.ContainerName()+"-db"))

	o := network.NewHTTPOptions("http://127.0.0.1")
	o.Timeout = 90
	o.Headers["Host"] = app.HostName()
	err = network.EnsureHTTPStatus(o)
	assert.NoError(err)
}
