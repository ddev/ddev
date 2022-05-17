package version

import (
	exec2 "github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/version_constants"
	"github.com/stretchr/testify/require"
	"os"
	"runtime"
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

var DdevBin = "ddev"

func TestGetVersionInfo(t *testing.T) {
	assert := asrt.New(t)

	if os.Getenv("DDEV_BINARY_FULLPATH") != "" {
		DdevBin = os.Getenv("DDEV_BINARY_FULLPATH")
	}

	// Run `ddev version` so we force download of docker-compose if we don't have one.
	_, err := exec2.RunHostCommand(DdevBin, "version")
	require.NoError(t, err)

	v := GetVersionInfo()

	assert.Equal(version_constants.DdevVersion, v["DDEV version"])
	assert.Contains(v["web"], version_constants.WebImg)
	assert.Contains(v["web"], version_constants.WebTag)
	assert.Contains(v["db"], version_constants.DBImg)
	assert.Contains(v["db"], nodeps.MariaDBDefaultVersion)
	assert.Contains(v["dba"], version_constants.DBAImg)
	assert.Contains(v["dba"], version_constants.DBATag)
	assert.Equal(runtime.GOOS, v["os"])
	assert.Equal(version_constants.BUILDINFO, v["build info"])
	assert.NotEmpty(v["docker-compose"])
	assert.NotEmpty(v["docker-platform"])

	assert.NotEmpty(v["docker"])
}
