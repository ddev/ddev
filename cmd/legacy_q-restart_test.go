package cmd

import (
	"fmt"
	"testing"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestLegacyStart runs drud legacy start on the test apps
func TestLegacyRestart(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)
	args := []string{"dev", "restart", LegacyTestApp, LegacyTestEnv}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	format := fmt.Sprintf
	assert.Contains(string(out), format("Stopping legacy-%s-%s-web ... done", LegacyTestApp, LegacyTestEnv))
	assert.Contains(string(out), format("Stopping legacy-%s-%s-db ... done", LegacyTestApp, LegacyTestEnv))
	assert.Contains(string(out), format("Starting legacy-%s-%s-web", LegacyTestApp, LegacyTestEnv))
	assert.Contains(string(out), format("Starting legacy-%s-%s-db", LegacyTestApp, LegacyTestEnv))
	assert.Contains(string(out), "Your application can be reached at")

	app := local.LegacyApp{}
	app.AppBase.Name = LegacyTestApp
	app.AppBase.Environment = LegacyTestEnv

	assert.Equal(true, utils.IsRunning(app.ContainerName()+"-web"))
	assert.Equal(true, utils.IsRunning(app.ContainerName()+"-db"))

	webPort, err := local.GetPodPort(app.ContainerName() + "-web")
	assert.NoError(err)
	dbPort, err := local.GetPodPort(app.ContainerName() + "-db")
	assert.NoError(err)

	assert.Equal(true, utils.IsTCPPortAvailable(int(webPort)))
	assert.Equal(true, utils.IsTCPPortAvailable(int(dbPort)))
	o := utils.NewHTTPOptions("http://127.0.0.1")
	o.Timeout = 420
	o.Headers["Host"] = app.HostName()
	err = utils.EnsureHTTPStatus(o)
	assert.NoError(err)
}
