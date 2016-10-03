package cmd

import (
	"fmt"
	"testing"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/bootstrap/cli/utils"
	drudutils "github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestLegacyStop runs drud legacy stop on the test apps
func TestLegacyStop(t *testing.T) {
	args := []string{"legacy", "stop", "-a", legacyTestApp}
	out, err := drudutils.RunCommand(drudBin, args)
	assert.NoError(t, err)
	format := fmt.Sprintf
	assert.Contains(t, string(out), format("Stopping legacy-%s-%s-web ... done", legacyTestApp, legacyTestEnv))
	assert.Contains(t, string(out), format("Stopping legacy-%s-%s-db ... done", legacyTestApp, legacyTestEnv))

}

// TestLegacyStart runs drud legacy start on the test apps
func TestLegacyStart(t *testing.T) {
	assert := assert.New(t)

	args := []string{"legacy", "start", "-a", legacyTestApp}
	out, err := drudutils.RunCommand(drudBin, args)
	assert.NoError(err)
	format := fmt.Sprintf
	assert.Contains(string(out), format("Starting legacy-%s-%s-web", legacyTestApp, legacyTestEnv))
	assert.Contains(string(out), format("Starting legacy-%s-%s-db", legacyTestApp, legacyTestEnv))
	assert.Contains(string(out), "WordPress site")

	app := local.LegacyApp{
		Name:        legacyTestApp,
		Environment: legacyTestEnv,
	}

	assert.Equal(true, utils.IsRunning(app.ContainerName()+"-web"))
	assert.Equal(true, utils.IsRunning(app.ContainerName()+"-db"))

	webPort, err := local.GetPodPort(app.ContainerName() + "-web")
	assert.NoError(err)
	dbPort, err := local.GetPodPort(app.ContainerName() + "-db")
	assert.NoError(err)

	assert.Equal(true, IsTCPPortAvailable(int(webPort)))
	assert.Equal(true, IsTCPPortAvailable(int(dbPort)))
	err = drudutils.EnsureHTTPStatus(fmt.Sprintf("http://localhost:%d", webPort), "", "", 120, 200)
	assert.NoError(err)
}
