package cmd

import "os"

var (
	binary = "ddev" // The ddev binary to use.
)

func setup() {
	if os.Getenv("DRUD_BINARY_FULLPATH") != "" {
		binary = os.Getenv("DRUD_BINARY_FULLPATH")
	}
}
