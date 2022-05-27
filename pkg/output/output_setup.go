package output

import (
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
	UserOut.Out = os.Stdout
	UserErr.Out = os.Stderr
	UserErr.SetOutput(&ErrorWriter{})

	if !JSONOutput {
		UserOut.Formatter = UserOutFormatter
		UserErr.Formatter = UserOutFormatter
	} else {
		UserOut.Formatter = &log.JSONFormatter{}
		UserErr.Formatter = &log.JSONFormatter{}
	}

	UserOutFormatter.DisableTimestamp = true
	// Use default log.InfoLevel for UserOut
	UserOut.Level = log.InfoLevel // UserOut will by default always output
	logLevel := log.InfoLevel

	// But we use custom DDEV_DEBUG-settable loglevel for log; export DDEV_DEBUG=true
	ddevDebug := os.Getenv("DDEV_DEBUG")
	if ddevDebug != "" {
		logLevel = log.DebugLevel
		UserOut.Level = log.DebugLevel
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
