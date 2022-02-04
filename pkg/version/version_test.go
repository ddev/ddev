package version

import (
	exec2 "github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/nodeps"
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

	assert.Equal(DdevVersion, v["ddev_version"])
	assert.Contains(v["web"], WebImg)
	assert.Contains(v["web"], WebTag)
	assert.Contains(v["db"], DBImg)
	assert.Contains(v["db"], nodeps.MariaDBDefaultVersion)
	assert.Contains(v["dba"], DBAImg)
	assert.Contains(v["dba"], DBATag)
	assert.Equal(runtime.GOOS, v["os"])
	assert.Equal(BUILDINFO, v["build_info"])
	assert.NotEmpty(v["docker_compose"])
	assert.NotEmpty(v["docker_platform"])

	assert.NotEmpty(v["docker"])
}
