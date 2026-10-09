package dockerutil

import (
	"testing"

	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSanitizeUsername tests the username sanitization logic
func TestSanitizeUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple username",
			input:    "john",
			expected: "john",
		},
		{
			name:     "Username with spaces",
			input:    "John Doe",
			expected: "johndoe",
		},
		{
			name:     "Username with @ symbol",
			input:    "user@example.com",
			expected: "userexamplecom",
		},
		{
			name:     "Username with diacritics (André)",
			input:    "André Kraus",
			expected: "andrekraus",
		},
		{
			name:     "Username with diacritics (Mück)",
			input:    "Mück",
			expected: "muck",
		},
		{
			name:     "Windows domain\\user format",
			input:    "DOMAIN\\JohnDoe",
			expected: "johndoe",
		},
		{
			name:     "Username with parentheses",
			input:    "John (Admin)",
			expected: "johnadmin",
		},
		{
			name:     "Username starting with number",
			input:    "310822",
			expected: "a310822",
		},
		{
			name:     "Username with multiple special characters",
			input:    "user!@#$%^&*()name",
			expected: "username",
		},
		{
			name:     "Username with underscores and hyphens (should be preserved)",
			input:    "user_name-123",
			expected: "user_name-123",
		},
		{
			name:     "Mixed case with special chars",
			input:    "JohnDoe@Company.COM",
			expected: "johndoecompanycom",
		},
		{
			name:     "Username with brackets",
			input:    "user[admin]",
			expected: "useradmin",
		},
		{
			name:     "Username with dots",
			input:    "john.doe",
			expected: "johndoe",
		},
		{
			name:     "Username with slashes",
			input:    "user/admin",
			expected: "useradmin",
		},
		{
			name:     "Empty string after sanitization",
			input:    "@@@",
			expected: "a",
		},
		{
			name:     "Only numbers",
			input:    "123456",
			expected: "a123456",
		},
		{
			name:     "Unicode characters",
			input:    "José García",
			expected: "josegarcia",
		},
		{
			name:     "Multiple backslashes (Windows path-like)",
			input:    "DOMAIN\\SUBDOMAIN\\User",
			expected: "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeUsername(tt.input)
			assert.Equal(t, tt.expected, result, "sanitizeUsername(%q) should return %q, got %q", tt.input, tt.expected, result)
		})
	}
}

// TestAddBoundHostPorts verifies that addBoundHostPorts extracts every non-empty
// HostPort from a network.PortMap, which is the logic GetBoundHostPorts relies on
// to fall back from HostConfig.PortBindings to NetworkSettings.Ports. Some
// providers (e.g. socktainer/Apple Container) report an empty HostConfig on
// inspect even though the container has ports bound, per NetworkSettings.
func TestAddBoundHostPorts(t *testing.T) {
	tests := []struct {
		name     string
		portMaps []network.PortMap
		expected map[string]bool
	}{
		{
			name:     "nil PortMap",
			portMaps: []network.PortMap{nil},
			expected: map[string]bool{},
		},
		{
			name: "single PortMap with bound ports",
			portMaps: []network.PortMap{
				{
					network.MustParsePort("80/tcp"): []network.PortBinding{{HostPort: "8080"}},
				},
			},
			expected: map[string]bool{"8080": true},
		},
		{
			name: "PortBinding with empty HostPort is excluded",
			portMaps: []network.PortMap{
				{
					network.MustParsePort("80/tcp"): []network.PortBinding{{HostPort: ""}},
				},
			},
			expected: map[string]bool{},
		},
		{
			name: "merging two PortMaps, as when falling back from HostConfig to NetworkSettings",
			portMaps: []network.PortMap{
				{},
				{
					network.MustParsePort("80/tcp"):  []network.PortBinding{{HostPort: "8080"}},
					network.MustParsePort("443/tcp"): []network.PortBinding{{HostPort: "8443"}},
				},
			},
			expected: map[string]bool{"8080": true, "8443": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			portMap := map[string]bool{}
			for _, m := range tt.portMaps {
				addBoundHostPorts(portMap, m)
			}
			require.Equal(t, tt.expected, portMap)
		})
	}
}

func TestMergePortBindings(t *testing.T) {
	bind := func(hostPort string) []network.PortBinding {
		return []network.PortBinding{{HostPort: hostPort}}
	}
	http, mailpit, vite := network.MustParsePort("80/tcp"), network.MustParsePort("8025/tcp"), network.MustParsePort("5173/tcp")
	tests := []struct {
		name       string
		configured network.PortMap
		actual     network.PortMap
		expected   network.PortMap
	}{
		{
			name:       "configured host ports win over actual",
			configured: network.PortMap{http: bind("8080")},
			actual:     network.PortMap{http: bind("8080")},
			expected:   network.PortMap{http: bind("8080")},
		},
		{
			name:       "stopped container keeps configured ports",
			configured: network.PortMap{http: bind("8080"), vite: bind("24117")},
			actual:     nil,
			expected:   network.PortMap{http: bind("8080"), vite: bind("24117")},
		},
		{
			name:       "empty configured host port takes the actual one",
			configured: network.PortMap{http: bind("8080"), mailpit: bind("8027"), vite: bind("")},
			actual:     network.PortMap{http: bind("8080"), mailpit: bind("8027"), vite: bind("55012")},
			expected:   network.PortMap{http: bind("8080"), mailpit: bind("8027"), vite: bind("55012")},
		},
		{
			name:       "empty PortBindings uses actual ports",
			configured: network.PortMap{},
			actual:     network.PortMap{http: bind("55001")},
			expected:   network.PortMap{http: bind("55001")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, mergePortBindings(tt.configured, tt.actual))
		})
	}
}
