package nodeps

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/term"
)

// ArrayContainsString returns true if slice contains element
func ArrayContainsString(slice []string, element string) bool {
	if slice == nil {
		return false
	}
	return !(PosString(slice, element) == -1)
}

// PosString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func PosString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// RemoveItemFromSlice returns a slice with item removed
// If the item does not exist, the slice is unchanged
// This is quite slow in the scheme of things, so shouldn't
// be used without examination
func RemoveItemFromSlice(slice []string, item string) []string {
	pos := PosString(slice, item)
	if pos != -1 {
		// Remove the element at index i from a.
		copy(slice[pos:], slice[pos+1:]) // Shift slice[pos+1:] left one index.
		slice[len(slice)-1] = ""         // Erase last element (write zero value).
		slice = slice[:len(slice)-1]     // Truncate slice.
	}
	return slice
}

// From https://www.calhoun.io/creating-random-strings-in-go/
// nolint: revive
var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// RandomString creates a random string with a set length
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// IsAppleSilicon returns true if running on mac M1
func IsAppleSilicon() bool {
	return runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"
}

// IsGitpod returns true if running on gitpod.io
func IsGitpod() bool {
	if os.Getenv("DDEV_PRETEND_GITPOD") == "true" {
		return true
	}
	return runtime.GOOS == "linux" && os.Getenv("GITPOD_WORKSPACE_ID") != ""
}

// IsCodespaces returns true if running on GitHub Codespaces
func IsCodespaces() bool {
	if os.Getenv("DDEV_PRETEND_CODESPACES") == "true" {
		return true
	}
	return runtime.GOOS == "linux" && os.Getenv("CODESPACES") == "true"
}

// GetWSLDistro returns the WSL2 distro name if on Linux
func GetWSLDistro() string {
	wslDistro := ""
	if runtime.GOOS == "linux" {
		wslDistro = os.Getenv("WSL_DISTRO_NAME")
	}
	return wslDistro
}

// IsLetter returns true if all chars in string are alpha
func IsLetter(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// IsInteger returns true if the string is integer
func IsInteger(s string) bool {
	_, err := strconv.ParseInt(s, 0, 64)
	return err == nil
}

// GetTerminalWidthHeight returns width, height if on terminal
// or 80, 0 if not. If we can't get terminal info, we'll assume 80x24
func GetTerminalWidthHeight() (int, int) {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			return width, height
		}
	}
	return 80, 24
}

// IsIPAddress returns true if ip is ipv4 or ipv6 address
func IsIPAddress(ip string) bool {
	if net.ParseIP(ip) != nil {
		return true
	}
	return false
}

// GrepStringInBuffer finds strings that match needle
func GrepStringInBuffer(buffer string, needle string) []string {
	re := regexp.MustCompilePOSIX(needle)
	// Fixed: OLD code used FindStringSubmatch() which only returned first match
	// Example: ddev-redis-php has multiple "require" statements but only first was found
	// NEW code uses FindAllString() to return all matches for proper validation
	matches := re.FindAllString(buffer, -1)
	return matches
}

// PathWithSlashesToArray returns an array of all possible paths separated by slashes out
// of a single one.
// i.e. path/to/file will return {"path", "path/to", "path/to/file"}
func PathWithSlashesToArray(path string) []string {
	var paths []string
	var partial string
	for _, p := range strings.Split(path, "/") {
		partial += p
		if len(partial) > 0 {
			paths = append(paths, partial)
		}
		partial += "/"
	}
	return paths
}

// ParseURL parses a URL and returns the scheme, URL without port, and port
// If the URL doesn't have an explicit port, returns the default port for the scheme
func ParseURL(rawURL string) (scheme string, urlWithoutPort string, port string) {
	if rawURL == "" {
		return "", "", ""
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", "", ""
	}

	// Check if the URL has a valid scheme
	scheme = parsedURL.Scheme
	if scheme != "http" && scheme != "https" {
		return "", "", ""
	}

	// Get URL without port
	urlWithoutPort = fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Hostname())

	// Get port
	if parsedURL.Port() != "" {
		port = parsedURL.Port()
	} else if parsedURL.Scheme == "https" {
		port = "443"
	} else {
		port = "80"
	}

	return scheme, urlWithoutPort, port
}
