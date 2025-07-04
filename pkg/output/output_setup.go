package output

import (
	"os"
	"strconv"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

type Fields = log.Fields

var (
	// UserOut is the customized logrus log used for direct user output
	UserOut = func() *log.Logger {
		l := log.New()
		l.SetOutput(os.Stdout)
		logLevel := log.InfoLevel
		if os.Getenv("DDEV_DEBUG") == "true" || os.Getenv("DDEV_VERBOSE") == "true" {
			logLevel = log.DebugLevel
		}
		l.SetLevel(logLevel)
		log.SetLevel(logLevel)
		if JSONOutput {
			l.SetFormatter(DdevOutputJSONFormatter)
			log.SetFormatter(DdevOutputJSONFormatter)
		} else {
			l.SetFormatter(DdevOutputFormatter)
			log.SetFormatter(DdevOutputFormatter)
		}
		return l
	}()
	// UserErr is the customized logrus log used for direct user stderr
	UserErr = func() *log.Logger {
		l := log.New()
		l.SetOutput(&ErrorWriter{})
		if JSONOutput {
			l.SetFormatter(DdevOutputJSONFormatter)
		} else {
			l.SetFormatter(DdevOutputFormatter)
		}
		return l
	}()
	// DdevOutputFormatter is the specialized formatter for UserOut
	DdevOutputFormatter = &TextFormatter{
		DisableColors:    !ColorsEnabled(),
		DisableTimestamp: true,
	}
	// DdevOutputJSONFormatter is the specialized JSON formatter for UserOut
	DdevOutputJSONFormatter = &log.JSONFormatter{}
	// JSONOutput indicates if JSON output mode is enabled, determined by command-line flags.
	// Parsed early, prior to Cobra flag initialization, to configure logging correctly from start.
	// Manual parsing is necessary because Cobra registers flags too late for this early use.
	JSONOutput = ParseBoolFlag("json-output", "j")
)

// ErrorWriter allows writing stderr
// Splitting to stderr approach from
// https://huynvk.dev/blog/4-tips-for-logging-on-gcp-using-golang-and-logrus
type ErrorWriter struct{}

func (w *ErrorWriter) Write(p []byte) (n int, err error) {
	return os.Stderr.Write(p)
}

// ParseBoolFlag scans os.Args backward to apply last-occurrence precedence for a boolean flag.
// Handles both --long[=true|false] and -s[=true|false] forms.
// Treats short flag in combined group (e.g. -xj) as implicit true.
// Returns false if the flag is absent or its value is invalid.
// Disabled entirely when running under `go test`.
func ParseBoolFlag(long string, short string) bool {
	if testing.Testing() {
		return false
	}
	args := os.Args[1:]
	longPrefix := "--" + long + "="
	shortPrefix := "-" + short + "="

	for i := len(args) - 1; i >= 0; i-- {
		arg := args[i]
		switch {
		case arg == "--"+long, arg == "-"+short:
			return true
		case strings.HasPrefix(arg, shortPrefix):
			v, err := strconv.ParseBool(arg[len(shortPrefix):])
			if err == nil {
				return v
			}
		case strings.HasPrefix(arg, longPrefix):
			v, err := strconv.ParseBool(arg[len(longPrefix):])
			if err == nil {
				return v
			}
		default:
			if len(arg) > 1 && arg[0] == '-' && arg[1] != '-' {
				for _, ch := range arg[1:] {
					if string(ch) == short {
						return true
					}
				}
			}
		}
	}
	return false
}

// ColorsEnabled returns true if colored output is enabled
// Implementation from https://no-color.org/
func ColorsEnabled() bool {
	return os.Getenv("NO_COLOR") == "" || os.Getenv("NO_COLOR") == "0"
}
