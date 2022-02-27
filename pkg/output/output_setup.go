package output

import (
	log "github.com/sirupsen/logrus"
	"os"
	"io"
)

var (
	// UserOut is the customized logrus log used for direct user output
	UserOut = log.New()
	// UserErr is the customized logrus log used for direct user stderr
	UserErr = log.New()
	// UserOutFormatter is the specialized formatter for UserOut
	UserOutFormatter = new(TextFormatter)
	// JSONOutput is a bool telling whether we're outputting in json. Set by command-line args.
	JSONOutput = false
	// NoOutput is a bool telling whether we want to supress any output. Set by command-line args.
	NoOutput = false
)

// LogSetUp sets up UserOut and log loggers as needed by ddev
func LogSetUp() {
	UserOut.Out = os.Stdout
	UserErr.Out = os.Stderr
	UserErr.SetOutput(&ErrorWriter{})

	if !JSONOutput {
		UserOut.Formatter = UserOutFormatter
		UserErr.Formatter = UserOutFormatter
	} else {
		UserOut.Formatter = &JSONFormatter{}
		UserErr.Formatter = &JSONFormatter{}
	}

	UserOutFormatter.DisableTimestamp = true
	// Always use log.DebugLevel for UserOut
	UserOut.Level = log.DebugLevel // UserOut will by default always output

	// But we use custom DDEV_DEBUG-settable loglevel for log
	logLevel := log.InfoLevel
	ddevDebug := os.Getenv("DDEV_DEBUG")
	if ddevDebug != "" {
		logLevel = log.DebugLevel
	}
	log.SetLevel(logLevel)

	if NoOutput {
		UserErr.Out = io.Discard
	}
}

// ErrorWriter allows writing stderr
// Splitting to stderr approach from
// https://huynvk.dev/blog/4-tips-for-logging-on-gcp-using-golang-and-logrus
type ErrorWriter struct{}

func (w *ErrorWriter) Write(p []byte) (n int, err error) {
	return os.Stderr.Write(p)
}
