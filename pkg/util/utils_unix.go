// +build !windows

package util

const (
	// BashBinary defines the binary name of bash.
	BashBinary = "bash"
)

// FindBashPath returns the bash binary. On Unix like systems only the name is
// returned, on Windows this will be the full path and name to the found binary.
func FindBashPath() (string, error) {
	return BashBinary, nil
}
