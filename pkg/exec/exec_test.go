package exec

import (
	"testing"

	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetHostShell checks that the host shell can be determined
func TestGetHostShell(t *testing.T) {
	assert := asrt.New(t)

	// Assert that the host shell is one of bash, fish, or zsh
	hostShell, err := GetHostShell()
	require.NoError(t, err)
	assert.Contains([]string{"bash", "fish", "zsh"}, hostShell)
}
