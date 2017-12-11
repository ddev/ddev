package cmd

import (
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/version"
	asrt "github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	assert := asrt.New(t)
	v := handleVersionCommand().String()
	output := strings.TrimSpace(v)
	assert.Contains(output, version.DdevVersion)
	assert.Contains(output, version.WebImg)
	assert.Contains(output, version.WebTag)
	assert.Contains(output, version.DBImg)
	assert.Contains(output, version.DBTag)
	assert.Contains(output, version.DBAImg)
	assert.Contains(output, version.DBATag)
	assert.Contains(output, version.DDevTLD)
}
