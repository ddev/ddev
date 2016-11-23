package cmd

import "os"

var (
	binary = "drud" // The drud binary to use.
)

func setup() {
	if os.Getenv("DRUD_BINARY_FULLPATH") != "" {
		binary = os.Getenv("DRUD_BINARY_FULLPATH")
	}
}
