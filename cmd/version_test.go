package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/drud/bootstrap/cli/version"
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
	assert.Contains(output, version.VERSION)
	assert.Contains(output, version.NGINX)
	assert.Contains(output, version.MYSQL)
}
