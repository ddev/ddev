package nodeps

import (
	asrt "github.com/stretchr/testify/assert"
	"testing"
)

// TestRandomString tests if RandomString returns the correct character length
func TestRandomString(t *testing.T) {
	randomString := RandomString(10)

	// is RandomString as long as required
	asrt.Equal(t, 10, len(randomString))
}

// TestPathWithSlashesToArray tests PathWithSlashesToArray
func TestPathWithSlashesToArray(t *testing.T) {
	assert := asrt.New(t)

	testSources := []string{
		"sites/default/files",
		"/sites/default/files",
		"./sites/default/files",
	}

	testExpectations := [][]string{
		{"sites", "sites/default", "sites/default/files"},
		{"/sites", "/sites/default", "/sites/default/files"},
		{".", "./sites", "./sites/default", "./sites/default/files"},
	}

	for i := 0; i < len(testSources); i++ {
		res := PathWithSlashesToArray(testSources[i])
		assert.Equal(testExpectations[i], res)
	}
}
