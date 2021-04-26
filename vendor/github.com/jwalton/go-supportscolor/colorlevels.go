package supportscolor

//go:generate stringer -type=ColorLevel

// ColorLevel represents the ANSI color level supported by the terminal.
type ColorLevel int

const (
	// None represents a terminal that does not support color at all.
	None ColorLevel = 0
	// Basic represents a terminal with basic 16 color support.
	Basic ColorLevel = 1
	// Ansi256 represents a terminal with 256 color support.
	Ansi256 ColorLevel = 2
	// Ansi16m represents a terminal with full true color support.
	Ansi16m ColorLevel = 3
)
