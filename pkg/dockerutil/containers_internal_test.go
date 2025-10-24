package dockerutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
