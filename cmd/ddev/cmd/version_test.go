package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/version"
	"github.com/stretchr/testify/assert"
)

func init() {
	setup()
}

func TestVersion(t *testing.T) {
	assert := assert.New(t)
	v, err := exec.Command(binary, "version").Output()
	assert.NoError(err)
	output := strings.TrimSpace(string(v))
	assert.Contains(output, version.DdevVersion)
	assert.Contains(output, version.WebImg)
	assert.Contains(output, version.WebTag)
	assert.Contains(output, version.DBImg)
	assert.Contains(output, version.DBTag)
}
