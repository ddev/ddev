package nodeps

import (
	"math/rand"
	"net"
	"os"
	"regexp"
	"runtime"
	"strconv"
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

// IsCodespaces returns true if running on Github Codespaces
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
// or 80, 0 if not. If we can't get terminal info, we'll just assume 80x24
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
	matches := re.FindStringSubmatch(buffer)
	return matches
}
