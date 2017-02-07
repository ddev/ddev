package cmd

import (
	"fmt"
	"testing"

	"github.com/drud/ddev/pkg/local"
	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestDevStop runs drud legacy stop on the test apps
func TestDevStop(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	args := []string{"stop", DevTestApp, DevTestEnv}
	out, err := utils.RunCommand(DdevBin, args)
	assert.NoError(t, err)
	format := fmt.Sprintf
	assert.Contains(t, string(out), format("Stopping legacy-%s-%s-web ... done", DevTestApp, DevTestEnv))
	assert.Contains(t, string(out), format("Stopping legacy-%s-%s-db ... done", DevTestApp, DevTestEnv))
}

func TestDevStoppedList(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	args := []string{"list"}
	out, err := utils.RunCommand(DdevBin, args)
	assert.NoError(t, err)
	assert.Contains(t, string(out), "found")
	assert.Contains(t, string(out), DevTestApp)
	assert.Contains(t, string(out), DevTestEnv)
	assert.Contains(t, string(out), "exited")
}

// TestDevStart runs drud legacy start on the test apps
func TestDevStart(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)

	args := []string{"start", DevTestApp, DevTestEnv}
	out, err := utils.RunCommand(DdevBin, args)
	assert.NoError(err)
	format := fmt.Sprintf
	assert.Contains(string(out), format("Starting legacy-%s-%s-web", DevTestApp, DevTestEnv))
	assert.Contains(string(out), format("Starting legacy-%s-%s-db", DevTestApp, DevTestEnv))
	assert.Contains(string(out), "Your application can be reached at")

	app := local.LegacyApp{}
	app.AppBase.Name = DevTestApp
	app.AppBase.Environment = DevTestEnv

	assert.Equal(true, utils.IsRunning(app.ContainerName()+"-web"))
	assert.Equal(true, utils.IsRunning(app.ContainerName()+"-db"))

	o := utils.NewHTTPOptions("http://127.0.0.1")
	o.Timeout = 90
	o.Headers["Host"] = app.HostName()
	err = utils.EnsureHTTPStatus(o)
	assert.NoError(err)
}
