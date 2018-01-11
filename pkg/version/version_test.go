package version

import (
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

func TestGetVersionInfo(t *testing.T) {
	assert := asrt.New(t)
	v := GetVersionInfo()
	assert.Contains(v, "cli")
	assert.Contains(v, "web")
	assert.Contains(v, "db")
	assert.Contains(v, "dba")
	assert.Contains(v, "router")
	assert.Contains(v, "commit")
	assert.Contains(v, "domain")
	assert.Contains(v, "build info")
	assert.Equal(DdevVersion, v["cli"])
	assert.Contains(v["web"], WebImg)
	assert.Contains(v["web"], WebTag)
	assert.Contains(v["db"], DBImg)
	assert.Contains(v["db"], DBTag)
	assert.Contains(v["dba"], DBAImg)
	assert.Contains(v["dba"], DBATag)
	assert.Equal(COMMIT, v["commit"])
	assert.Equal(DDevTLD, v["domain"])
	assert.Equal(BUILDINFO, v["build info"])
}
