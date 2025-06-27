package output

import (
	log "github.com/sirupsen/logrus"
	"os"
	"slices"
)

type Fields = log.Fields

var (
	// UserOut is the customized logrus log used for direct user output
	UserOut = func() *log.Logger {
		l := log.New()
		l.SetOutput(os.Stdout)
		logLevel := log.InfoLevel
		if os.Getenv("DDEV_DEBUG") != "" {
			logLevel = log.DebugLevel
		}
		l.SetLevel(logLevel)
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
	JSONOutput = func() bool {
		args := os.Args[1:]
		return slices.Contains(args, "-j") || slices.Contains(args, "--json-output")
	}()
)

// ErrorWriter allows writing stderr
// Splitting to stderr approach from
// https://huynvk.dev/blog/4-tips-for-logging-on-gcp-using-golang-and-logrus
type ErrorWriter struct{}

func (w *ErrorWriter) Write(p []byte) (n int, err error) {
	return os.Stderr.Write(p)
}
