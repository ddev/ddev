package ddevapp_test

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
	"time"
)

// Test to make sure that the user provisioned in the container has
// the proper uid/gid/username characteristics.
func TestUserProvisioningInContainer(t *testing.T) {
	assert := asrt.New(t)
	app := &ddevapp.DdevApp{}

	// Make sure this leaves us in the original test directory
	testDir, _ := os.Getwd()
	//nolint: errcheck
	defer os.Chdir(testDir)

	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := util.TimeTrack(time.Now(), fmt.Sprintf("%s %s", site.Name, t.Name()))
	defer runTime()

	err := app.Init(site.Dir)
	assert.NoError(err)

	err = app.Start()
	assert.NoError(err)
	//nolint: errcheck
	defer app.Stop(true, false)

	// make sure files get created in the user?

	uid, gid, username := util.GetContainerUIDGid()

	for _, service := range []string{"web", "db"} {

		out, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: service,
			Cmd:     "id -un",
		})
		assert.NoError(err)
		assert.Equal(username, strings.Trim(out, "\n"))

		out, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: service,
			Cmd:     "id -u",
		})
		assert.NoError(err)
		assert.Equal(uid, strings.Trim(out, "\n"))

		out, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: service,
			Cmd:     "id -g",
		})
		assert.NoError(err)
		assert.Equal(gid, strings.Trim(out, "\n"))
	}

}
