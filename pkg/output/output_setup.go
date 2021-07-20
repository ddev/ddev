package output

import (
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"os"
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
)

// LogSetUp sets up UserOut and log loggers as needed by ddev
func LogSetUp() {
	// Use color.Output instead of stderr for all user output
	UserOut.Out = color.Output
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
	drudDebug := os.Getenv("DDEV_DEBUG")
	if drudDebug != "" {
		logLevel = log.DebugLevel
	}
	log.SetLevel(logLevel)
}

// ErrorWriter allows writing stderr
// Splitting to stderr approach from
// https://huynvk.dev/blog/4-tips-for-logging-on-gcp-using-golang-and-logrus
type ErrorWriter struct{}

func (w *ErrorWriter) Write(p []byte) (n int, err error) {
	return os.Stderr.Write(p)
}
