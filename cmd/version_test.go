package cmd

import (
	"os/exec"
	"strings"
	"testing"

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
	assert.Contains(output, cliVersion)
}
