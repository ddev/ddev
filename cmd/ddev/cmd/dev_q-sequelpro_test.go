package cmd

import (
	"path"
	"testing"

	"github.com/drud/ddev/pkg/local"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

// TestDevSequelproScaffold tests the the sequelpro config file is created successfully
func TestDevSequelproScaffold(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"sequelpro", DevTestApp, DevTestEnv, "-s"}
	out, err := system.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "finished successfully")

	app := local.LegacyApp{}
	app.AppBase.Name = DevTestApp
	app.AppBase.Environment = DevTestEnv

	assert.Equal(true, system.FileExists(path.Join(app.AbsPath(), "sequelpro.spf")))

}
