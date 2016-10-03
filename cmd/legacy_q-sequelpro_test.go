package cmd

import (
	"path"
	"testing"

	"github.com/drud/bootstrap/cli/local"
	drudutils "github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

// TestLegacySequelproScaffold tests the the sequelpro config file is created successfully
func TestLegacySequelproScaffold(t *testing.T) {
	assert := assert.New(t)

	// test that you get an error when you run with no args
	args := []string{"legacy", "sequelpro", "-a", legacyTestApp, "-s"}
	out, err := drudutils.RunCommand(drudBin, args)
	assert.NoError(err)
	assert.Contains(string(out), "finished successfully")

	app := local.LegacyApp{
		Name:        legacyTestApp,
		Environment: legacyTestEnv,
	}

	assert.Equal(true, drudutils.FileExists(path.Join(app.AbsPath(), "sequelpro.spf")))

}
