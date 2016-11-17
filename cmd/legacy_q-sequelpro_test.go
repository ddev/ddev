package cmd

import (
	"path"
	"testing"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestLegacySequelproScaffold tests the the sequelpro config file is created successfully
func TestLegacySequelproScaffold(t *testing.T) {
	if skipComposeTests {
		t.Skip("Compose tests being skipped.")
	}
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"legacy", "sequelpro", LegacyTestApp, LegacyTestEnv, "-s"}
	out, err := utils.RunCommand(DrudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "finished successfully")

	app := local.LegacyApp{
		Name:        LegacyTestApp,
		Environment: LegacyTestEnv,
	}

	assert.Equal(true, utils.FileExists(path.Join(app.AbsPath(), "sequelpro.spf")))

}
