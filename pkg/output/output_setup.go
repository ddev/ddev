package output

import (
	"github.com/drud/ddev/pkg/version"
	"github.com/evalphobia/logrus_sentry"
	"os"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
)

var (
	// UserOut is the customized logrus log used for direct user output
	UserOut = log.New()
	// UserOutFormatter is the specialized formatter for UserOut
	UserOutFormatter = new(TextFormatter)
	// JSONOutput is a bool telling whether we're outputting in json. Set by command-line args.
	JSONOutput = false
	// SentryDSN is the ddev-specific key for the Sentry service.
	SentryDSN = "https://ad3abb1deb8447398c5a2ad8f4287fad:70e11b442a9243719f150e4d922cfde6@sentry.io/160826"
)

// LogSetUp sets up UserOut and log loggers as needed by ddev
func LogSetUp() {
	// Use color.Output instead of stderr for all user output
	log.SetOutput(color.Output)
	UserOut.Out = color.Output

	tags := map[string]string{
		"commit": version.COMMIT,
	}
	levels := []log.Level{
		log.PanicLevel,
		log.FatalLevel,
		log.ErrorLevel,
	}

	// Report errors and panics to Sentry
	hook, err := logrus_sentry.NewAsyncWithTagsSentryHook(SentryDSN, tags, levels)

	if err == nil {
		UserOut.Hooks.Add(hook)
	}

	if !JSONOutput {
		UserOut.Formatter = UserOutFormatter
	} else {
		UserOut.Formatter = &JSONFormatter{}
	}

	UserOutFormatter.DisableTimestamp = true
	// Always use log.DebugLevel for UserOut
	UserOut.Level = log.DebugLevel // UserOut will by default always output

	// But we use custom DRUD_DEBUG-settable loglevel for log
	logLevel := log.InfoLevel
	drudDebug := os.Getenv("DRUD_DEBUG")
	if drudDebug != "" {
		logLevel = log.DebugLevel
	}
	log.SetLevel(logLevel)
}
