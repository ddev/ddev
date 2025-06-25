package output

import (
	log "github.com/sirupsen/logrus"
	"os"
)

type Fields = log.Fields

var (
	// UserOut is the customized logrus log used for direct user output
	UserOut = func() *log.Logger {
		l := log.New()
		l.SetFormatter(DdevOutputFormatter)
		l.SetOutput(os.Stdout)
		logLevel := log.InfoLevel
		if os.Getenv("DDEV_DEBUG") != "" {
			logLevel = log.DebugLevel
		}
		l.SetLevel(logLevel)
		return l
	}()
	// UserErr is the customized logrus log used for direct user stderr
	UserErr = func() *log.Logger {
		l := log.New()
		l.SetFormatter(DdevOutputFormatter)
		l.SetOutput(&ErrorWriter{})
		return l
	}()
	// DdevOutputFormatter is the specialized formatter for UserOut
	DdevOutputFormatter = &TextFormatter{
		// TODO: add DisableColors handler in a different PR
		DisableTimestamp: true,
	}
	// DdevOutputJSONFormatter is the specialized JSON formatter for UserOut
	DdevOutputJSONFormatter = &log.JSONFormatter{}
	// JSONOutput is a bool telling whether we're outputting in json. Set by command-line args.
	JSONOutput = false
)

// LogSetUp sets up UserOut and log loggers as needed by ddev
func LogSetUp() {
	// We don't use logrus directly in our code, but configure it here anyway
	log.SetFormatter(DdevOutputFormatter)
	log.SetLevel(UserOut.GetLevel())

	if JSONOutput {
		UserOut.SetFormatter(DdevOutputJSONFormatter)
		UserErr.SetFormatter(DdevOutputJSONFormatter)
		log.SetFormatter(DdevOutputJSONFormatter)
	}
}

// ErrorWriter allows writing stderr
// Splitting to stderr approach from
// https://huynvk.dev/blog/4-tips-for-logging-on-gcp-using-golang-and-logrus
type ErrorWriter struct{}

func (w *ErrorWriter) Write(p []byte) (n int, err error) {
	return os.Stderr.Write(p)
}
