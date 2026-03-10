package ddevapp

import (
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

// TestProviderInfoField tests that the InfoCommand field exists in ProviderInfo
func TestProviderInfoField(t *testing.T) {
	assert := asrt.New(t)

	// Test that InfoCommand field is properly defined in ProviderInfo
	p := Provider{
		ProviderInfo: ProviderInfo{
			InfoCommand: ProviderCommand{
				Command: "echo test",
				Service: "web",
			},
		},
	}

	assert.Equal("echo test", p.InfoCommand.Command)
	assert.Equal("web", p.InfoCommand.Service)
}
