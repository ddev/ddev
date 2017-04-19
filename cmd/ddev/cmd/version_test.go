package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/version"
	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	assert := assert.New(t)
	v, err := exec.Command(DdevBin, "version").Output()
	assert.NoError(err)
	output := strings.TrimSpace(string(v))
	assert.Contains(output, version.DdevVersion)
	assert.Contains(output, version.WebImg)
	assert.Contains(output, version.WebTag)
	assert.Contains(output, version.DBImg)
	assert.Contains(output, version.DBTag)
	assert.Contains(output, version.DBAImg)
	assert.Contains(output, version.DBATag)
}
