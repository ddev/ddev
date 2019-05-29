package version

import (
	"runtime"
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

func TestGetVersionInfo(t *testing.T) {
	assert := asrt.New(t)
	v := GetVersionInfo()

	assert.Equal(DdevVersion, v["DDEV-Local version"])
	assert.Contains(v["web"], WebImg)
	assert.Contains(v["web"], WebTag)
	assert.Contains(v["db"], DBImg)
	assert.Contains(v["db"], MariaDBDefaultVersion)
	assert.Contains(v["dba"], DBAImg)
	assert.Contains(v["dba"], DBATag)
	assert.Equal(COMMIT, v["commit"])
	assert.Equal(runtime.GOOS, v["os"])
	assert.Equal(BUILDINFO, v["build info"])
	assert.NotEmpty(v["docker-compose"])
	assert.NotEmpty(v["docker"])
}
