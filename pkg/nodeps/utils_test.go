package nodeps_test

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestRandomString tests if RandomString returns the correct character length
func TestRandomString(t *testing.T) {
	randomString := nodeps.RandomString(10)

	// is RandomString as long as required
	require.Equal(t, 10, len(randomString))
}

// TestPathWithSlashesToArray tests PathWithSlashesToArray
func TestPathWithSlashesToArray(t *testing.T) {
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
		res := nodeps.PathWithSlashesToArray(testSources[i])
		require.Equal(t, testExpectations[i], res)
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
			scheme, url, port := nodeps.ParseURL(tc.url)
			require.Equal(t, tc.expectedScheme, scheme, "scheme should match for %s", tc.url)
			require.Equal(t, tc.expectedURL, url, "URL without port should match for %s", tc.url)
			require.Equal(t, tc.expectedPort, port, "port should match for %s", tc.url)
		})
	}
}

// TestArrayContainsString tests ArrayContainsString function
func TestArrayContainsString(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	require.True(t, nodeps.ArrayContainsString(slice, "banana"))
	require.False(t, nodeps.ArrayContainsString(slice, "grape"))
	require.False(t, nodeps.ArrayContainsString([]string{}, "apple"))
	require.False(t, nodeps.ArrayContainsString(nil, "apple"))
}

// TestPosString tests PosString function
func TestPosString(t *testing.T) {
	slice := []string{"apple", "banana", "cherry", "banana"}

	require.Equal(t, 1, nodeps.PosString(slice, "banana"))
	require.Equal(t, 0, nodeps.PosString(slice, "apple"))
	require.Equal(t, 2, nodeps.PosString(slice, "cherry"))
	require.Equal(t, -1, nodeps.PosString(slice, "grape"))
	require.Equal(t, -1, nodeps.PosString([]string{}, "apple"))
	require.Equal(t, -1, nodeps.PosString(nil, "apple"))
}

// TestRemoveItemFromSlice tests RemoveItemFromSlice function
func TestRemoveItemFromSlice(t *testing.T) {
	// Test removing existing item
	slice := []string{"apple", "banana", "cherry"}
	result := nodeps.RemoveItemFromSlice(slice, "banana")
	expected := []string{"apple", "cherry"}
	require.Equal(t, expected, result)

	// Test removing non-existing item
	slice2 := []string{"apple", "banana", "cherry"}
	result2 := nodeps.RemoveItemFromSlice(slice2, "grape")
	expected2 := []string{"apple", "banana", "cherry"}
	require.Equal(t, expected2, result2)

	// Test removing from empty slice
	result3 := nodeps.RemoveItemFromSlice([]string{}, "apple")
	require.Equal(t, []string{}, result3)

	// Test removing first item
	slice4 := []string{"apple", "banana", "cherry"}
	result4 := nodeps.RemoveItemFromSlice(slice4, "apple")
	expected4 := []string{"banana", "cherry"}
	require.Equal(t, expected4, result4)

	// Test removing last item
	slice5 := []string{"apple", "banana", "cherry"}
	result5 := nodeps.RemoveItemFromSlice(slice5, "cherry")
	expected5 := []string{"apple", "banana"}
	require.Equal(t, expected5, result5)
}

// TestIsMacOS tests IsMacOS function
func TestIsMacOS(t *testing.T) {
	result := nodeps.IsMacOS()
	require.IsType(t, false, result)

	if runtime.GOOS == "darwin" {
		require.True(t, result, "Should be true on macOS (darwin)")
	} else {
		require.False(t, result, "Should be false on non-macOS platforms")
	}
}

// TestIsAppleSilicon tests IsAppleSilicon function
func TestIsAppleSilicon(t *testing.T) {
	// Test that the function returns a boolean without error
	result := nodeps.IsAppleSilicon()
	require.IsType(t, false, result)

	// Test the logic: should only be true on darwin/arm64
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		require.True(t, result, "Should be true on Apple Silicon (darwin/arm64)")
	} else {
		require.False(t, result, "Should be false on non-Apple Silicon platforms")
	}
}

// TestIsWindows tests IsWindows function
func TestIsWindows(t *testing.T) {
	result := nodeps.IsWindows()
	require.IsType(t, false, result)

	if runtime.GOOS == "windows" {
		require.True(t, result, "Should be true on Windows")
	} else {
		require.False(t, result, "Should be false on non-Windows platforms")
	}
}

// TestIsLinux tests IsLinux function
func TestIsLinux(t *testing.T) {
	result := nodeps.IsLinux()
	require.IsType(t, false, result)

	if runtime.GOOS == "linux" {
		require.True(t, result, "Should be true on Linux")
	} else {
		require.False(t, result, "Should be false on non-Linux platforms")
	}
}

// TestIsWSL2 tests IsWSL2 function
func TestIsWSL2(t *testing.T) {
	result := nodeps.IsWSL2()
	require.IsType(t, false, result)

	// IsWSL2 should only return true if:
	// 1. Running on Linux, AND
	// 2. Either WSL_INTEROP environment variable is set, OR
	// 3. /proc/version contains "-microsoft"
	if runtime.GOOS != "linux" {
		require.False(t, result, "Should be false on non-Linux platforms")
	} else {
		// On Linux, check if we're actually in WSL2
		wslInterop := os.Getenv("WSL_INTEROP") != ""
		procVersionContainsMicrosoft := false

		if procVersionBytes, err := os.ReadFile("/proc/version"); err == nil {
			procVersionContainsMicrosoft = strings.Contains(string(procVersionBytes), "-microsoft")
		}

		expectedResult := wslInterop || procVersionContainsMicrosoft
		require.Equal(t, expectedResult, result, "IsWSL2 result should match expected logic")
	}
}

// TestIsCodespaces tests IsCodespaces function
func TestIsCodespaces(t *testing.T) {
	// Test that the function returns a boolean without error
	result := nodeps.IsCodespaces()
	require.IsType(t, false, result)

	// Test the logic based on current environment
	// IsCodespaces should return true only if:
	// 1. DDEV_PRETEND_CODESPACES=true (for testing purposes), OR
	// 2. Running on Linux AND CODESPACES=true
	pretendCodespaces := os.Getenv("DDEV_PRETEND_CODESPACES") == "true"
	codespacesEnv := os.Getenv("CODESPACES") == "true"
	isLinux := runtime.GOOS == "linux"

	expectedResult := pretendCodespaces || (isLinux && codespacesEnv)
	require.Equal(t, expectedResult, result)
}

// TestGetWSLDistro tests GetWSLDistro function
func TestGetWSLDistro(t *testing.T) {
	// Test that the function returns a string without error
	result := nodeps.GetWSLDistro()
	require.IsType(t, "", result)

	// Test the logic based on current environment
	// GetWSLDistro should return WSL_DISTRO_NAME only if running on Linux
	wslDistroName := os.Getenv("WSL_DISTRO_NAME")
	isLinux := runtime.GOOS == "linux"

	if isLinux {
		require.Equal(t, wslDistroName, result)
	} else {
		require.Equal(t, "", result)
	}
}

// TestIsLetter tests IsLetter function
func TestIsLetter(t *testing.T) {
	require.True(t, nodeps.IsLetter("abc"))
	require.True(t, nodeps.IsLetter("ABC"))
	require.True(t, nodeps.IsLetter("aBc"))
	require.True(t, nodeps.IsLetter(""))
	require.False(t, nodeps.IsLetter("abc123"))
	require.False(t, nodeps.IsLetter("123"))
	require.False(t, nodeps.IsLetter("abc!"))
	require.False(t, nodeps.IsLetter("a b"))
	require.False(t, nodeps.IsLetter("a-b"))
}

// TestIsInteger tests IsInteger function
func TestIsInteger(t *testing.T) {
	require.True(t, nodeps.IsInteger("123"))
	require.True(t, nodeps.IsInteger("-123"))
	require.True(t, nodeps.IsInteger("0"))
	require.True(t, nodeps.IsInteger("0x10")) // hex
	require.True(t, nodeps.IsInteger("010"))  // octal
	require.False(t, nodeps.IsInteger("123.45"))
	require.False(t, nodeps.IsInteger("abc"))
	require.False(t, nodeps.IsInteger("12a"))
	require.False(t, nodeps.IsInteger(""))
	require.False(t, nodeps.IsInteger(" 123 "))
}

// TestIsIPAddress tests IsIPAddress function
func TestIsIPAddress(t *testing.T) {
	// Valid IPv4 addresses
	require.True(t, nodeps.IsIPAddress("192.168.1.1"))
	require.True(t, nodeps.IsIPAddress("127.0.0.1"))
	require.True(t, nodeps.IsIPAddress("0.0.0.0"))
	require.True(t, nodeps.IsIPAddress("255.255.255.255"))

	// Valid IPv6 addresses
	require.True(t, nodeps.IsIPAddress("::1"))
	require.True(t, nodeps.IsIPAddress("2001:db8::1"))
	require.True(t, nodeps.IsIPAddress("fe80::1"))

	// Invalid addresses
	require.False(t, nodeps.IsIPAddress(""))
	require.False(t, nodeps.IsIPAddress("localhost"))
	require.False(t, nodeps.IsIPAddress("256.256.256.256"))
	require.False(t, nodeps.IsIPAddress("192.168.1"))
	require.False(t, nodeps.IsIPAddress("192.168.1.1.1"))
	require.False(t, nodeps.IsIPAddress("not-an-ip"))
}

// TestGrepStringInBuffer tests GrepStringInBuffer function with multiple matches
func TestGrepStringInBuffer(t *testing.T) {
	// Test single match
	buffer1 := "This is a test string with one match"
	matches1 := nodeps.GrepStringInBuffer(buffer1, "test")
	require.Equal(t, []string{"test"}, matches1)

	// Test multiple matches - this was the main issue that was fixed
	buffer2 := `require_once 'config.php';
require_once 'database.php';
include 'header.php';
require 'footer.php';
require_once 'utils.php';`
	matches2 := nodeps.GrepStringInBuffer(buffer2, `^(require|require_once).*\.php';`)
	expected2 := []string{"require_once 'config.php';", "require_once 'database.php';", "require 'footer.php';", "require_once 'utils.php';"}
	require.Equal(t, expected2, matches2)

	// Test no matches
	buffer4 := "This string has no matching pattern"
	matches4 := nodeps.GrepStringInBuffer(buffer4, "xyz")
	require.Equal(t, []string(nil), matches4)

	// Test empty buffer
	matches5 := nodeps.GrepStringInBuffer("", "test")
	require.Equal(t, []string(nil), matches5)

	// Test with line breaks and multiple matches
	buffer6 := "Line 1: apple\nLine 2: banana\nLine 3: apple pie\nLine 4: pineapple"
	matches6 := nodeps.GrepStringInBuffer(buffer6, "apple")
	expected6 := []string{"apple", "apple", "apple"}
	require.Equal(t, expected6, matches6)

	// Test case sensitivity
	buffer7 := "Test TEST test TeSt"
	matches7 := nodeps.GrepStringInBuffer(buffer7, "test")
	expected7 := []string{"test"}
	require.Equal(t, expected7, matches7)

	// Test with special regex characters
	buffer8 := "price: $10.50, discount: $5.25, total: $15.75"
	matches8 := nodeps.GrepStringInBuffer(buffer8, `\$[0-9]+\.[0-9]+`)
	expected8 := []string{"$10.50", "$5.25", "$15.75"}
	require.Equal(t, expected8, matches8)
}
