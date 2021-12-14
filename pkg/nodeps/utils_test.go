package nodeps

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestRandomString tests if RandomString returns the correct character length
func TestRandomString(t *testing.T) {
	randomString := RandomString(10)

	// is RandomString as long as required
	assert.Equal(t, 10, len(randomString))
}
