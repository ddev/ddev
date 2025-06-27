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
		// TODO: add DisableColors handler in a different PR
		DisableTimestamp: true,
	}
	// DdevOutputJSONFormatter is the specialized JSON formatter for UserOut
	DdevOutputJSONFormatter = &log.JSONFormatter{}
	// JSONOutput is a bool telling whether we're outputting in JSON. Set by command-line args.
	// Parsed early (before Cobra) because logging initialization depends on this value.
	JSONOutput = func() bool {
		for _, arg := range os.Args[1:] {
			switch arg {
			case "-j", "--json-output": // Standalone flags
				return true
			case "-j=true", "-j=1", "--json-output=true", "--json-output=1": // Explicit true/1 values
				return true
			}
		}
		return false
	}()
)

// ErrorWriter allows writing stderr
// Splitting to stderr approach from
// https://huynvk.dev/blog/4-tips-for-logging-on-gcp-using-golang-and-logrus
type ErrorWriter struct{}

func (w *ErrorWriter) Write(p []byte) (n int, err error) {
	return os.Stderr.Write(p)
}
