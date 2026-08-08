package ddevapp_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContainerTerminfo checks that the containers you can ssh into resolve
// every TERM in nodeps.ContainerTerminfoEntries. TerminalExecEnv forwards those
// unchanged, so an entry missing from an image would give the shell an unusable
// TERM.
func TestContainerTerminfo(t *testing.T) {
	assert := asrt.New(t)
	site := TestSites[0]
	runTime := util.TimeTrackC(t.Name())

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)

	t.Cleanup(func() {
		runTime()
		err = app.Stop(true, false)
		assert.NoError(err)
	})

	err = app.Start()
	require.NoError(t, err)

	for _, service := range []string{"web", "db"} {
		stdout, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: service,
			Cmd: fmt.Sprintf(`for terminal in %s; do infocmp -- "$terminal" >/dev/null 2>&1 || echo "$terminal"; done`,
				strings.Join(nodeps.ContainerTerminfoEntries, " ")),
		})
		require.NoError(t, err)
		require.Empty(t, strings.TrimSpace(stdout), "%s container has no terminfo entry for: %s", service, stdout)
	}
}
