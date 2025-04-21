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

// TestParseURL tests the ParseURL function
func TestParseURL(t *testing.T) {
	tests := map[string]struct {
		url            string
		expectedScheme string
		expectedURL    string
		expectedPort   string
	}{
		"http URL without port": {
			url:            "http://example.com",
			expectedScheme: "http",
			expectedURL:    "http://example.com",
			expectedPort:   "80",
		},
		"https URL without port": {
			url:            "https://example.com",
			expectedScheme: "https",
			expectedURL:    "https://example.com",
			expectedPort:   "443",
		},
		"http URL with port": {
			url:            "http://example.com:8080",
			expectedScheme: "http",
			expectedURL:    "http://example.com",
			expectedPort:   "8080",
		},
		"https URL with port": {
			url:            "https://example.com:8443",
			expectedScheme: "https",
			expectedURL:    "https://example.com",
			expectedPort:   "8443",
		},
		"empty URL": {
			url:            "",
			expectedScheme: "",
			expectedURL:    "",
			expectedPort:   "",
		},
		"invalid URL": {
			url:            "not-a-url",
			expectedScheme: "",
			expectedURL:    "",
			expectedPort:   "",
		},
		"URL with path": {
			url:            "https://example.com/path",
			expectedScheme: "https",
			expectedURL:    "https://example.com",
			expectedPort:   "443",
		},
		"URL with query": {
			url:            "https://example.com?query=value",
			expectedScheme: "https",
			expectedURL:    "https://example.com",
			expectedPort:   "443",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := asrt.New(t)
			scheme, url, port := ParseURL(tc.url)
			assert.Equal(tc.expectedScheme, scheme, "scheme should match for %s", tc.url)
			assert.Equal(tc.expectedURL, url, "URL without port should match for %s", tc.url)
			assert.Equal(tc.expectedPort, port, "port should match for %s", tc.url)
		})
	}
}
