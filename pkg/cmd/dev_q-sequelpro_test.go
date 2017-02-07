package cmd

import (
	"path"
	"testing"

	"github.com/drud/ddev/local"
	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestDevSequelproScaffold tests the the sequelpro config file is created successfully
func TestDevSequelproScaffold(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"dev", "sequelpro", DevTestApp, DevTestEnv, "-s"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "finished successfully")

	app := local.LegacyApp{}
	app.AppBase.Name = DevTestApp
	app.AppBase.Environment = DevTestEnv

	assert.Equal(true, utils.FileExists(path.Join(app.AbsPath(), "sequelpro.spf")))

}
