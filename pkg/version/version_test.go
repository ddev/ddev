package version

import (
	"runtime"
	"testing"

	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testsetup"
	"github.com/ddev/ddev/pkg/versionconstants"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var DdevBin = "ddev"

func init() {
	DdevBin = testsetup.MustResolveDdevBinary()
}

func TestGetVersionInfo(t *testing.T) {
	assert := asrt.New(t)

	v, err := GetVersionInfo()
	require.NoError(t, err)

	assert.Equal(versionconstants.DdevVersion, v["DDEV version"])
	assert.Contains(v["web"], versionconstants.WebImg)
	assert.Contains(v["web"], docker.ResolveImageTag(versionconstants.WebTag, versionconstants.WebTagBranch))
	assert.Contains(v["db"], versionconstants.DBImg)
	assert.Contains(v["db"], nodeps.MariaDBDefaultVersion)
	assert.Equal(runtime.Version(), v["go-version"])
	assert.Equal(runtime.GOOS, v["os"])
	assert.Equal(versionconstants.BUILDINFO, v["build info"])
	assert.NotEmpty(v["docker"])
	assert.NotEmpty(v["docker-api"])
	assert.NotEmpty(v["docker-buildx"])
	assert.NotEmpty(v["docker-platform"])
	_, ok := v["image-tag-branches"]
	assert.False(ok, "image-tag-branches should not appear in GetVersionInfo(), it's display-only")
}

func TestImageTagBranches(t *testing.T) {
	assert := asrt.New(t)

	branches := ImageTagBranches()

	assert.Equal(versionconstants.WebTagBranch, branches["web"])
	assert.Equal(versionconstants.BaseDBTagBranch, branches["db"])
	assert.Equal(versionconstants.TraefikRouterTagBranch, branches["router"])
	assert.Equal(versionconstants.SSHAuthTagBranch, branches["ddev-ssh-agent"])
	assert.Equal(versionconstants.XhguiTagBranch, branches["xhgui-image"])
}
