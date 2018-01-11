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
	assert.Contains(WebImg, v["web"])
	assert.Contains(WebTag, v["web"])
	assert.Contains(DBImg, v["db"])
	assert.Contains(DBTag, v["db"])
	assert.Contains(DBAImg, v["dba"])
	assert.Contains(DBATag, v["dba"])
	assert.Equal(COMMIT, v["commit"])
	assert.Equal(DDevTLD, v["domain"])
	assert.Equal(BUILDINFO, v["build info"])
}
