package version

import (
	exec2 "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/versionconstants"
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

	assert.Equal(versionconstants.DdevVersion, v["DDEV version"])
	assert.Contains(v["web"], versionconstants.WebImg)
	assert.Contains(v["web"], versionconstants.WebTag)
	assert.Contains(v["db"], versionconstants.DBImg)
	assert.Contains(v["db"], nodeps.MariaDBDefaultVersion)
	assert.Equal(runtime.GOOS, v["os"])
	assert.Equal(versionconstants.BUILDINFO, v["build info"])
	assert.NotEmpty(v["docker-compose"])
	assert.NotEmpty(v["docker-platform"])

	assert.NotEmpty(v["docker"])
}
