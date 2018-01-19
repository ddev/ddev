package version

import (
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

func TestGetVersionInfo(t *testing.T) {
	assert := asrt.New(t)
	v := GetVersionInfo()

	assert.Equal(DdevVersion, v["cli"])
	assert.Contains(v["web"], WebImg)
	assert.Contains(v["db"], DBImg)
	assert.Contains(v["dba"], DBAImg)
	assert.Equal(COMMIT, v["commit"])
	assert.Equal(DDevTLD, v["domain"])
	assert.Equal(BUILDINFO, v["build info"])
}
