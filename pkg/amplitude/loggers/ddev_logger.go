package loggers

import (
	"fmt"
	"regexp"

	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

func NewDdevLogger(verbose bool) types.Logger {
	return &ddevLogger{
		verbose: verbose,
	}
}

type ddevLogger struct {
	verbose bool
}

func (l *ddevLogger) Debugf(message string, args ...interface{}) {
	if l.verbose {
		output.UserErr.Print(filterMessage(message, args...))
	}
}

func (l *ddevLogger) Infof(message string, args ...interface{}) {
	if l.verbose {
		output.UserErr.Info(filterMessage(util.ColorizeText(message, "green"), args...))
	}
}

func (l *ddevLogger) Warnf(message string, args ...interface{}) {
	if l.verbose {
		output.UserErr.Warn(filterMessage(util.ColorizeText(message, "yellow"), args...))
	}
}

func (l *ddevLogger) Errorf(message string, args ...interface{}) {
	if l.verbose {
		output.UserErr.Error(filterMessage(util.ColorizeText(message, "red"), args...))
	}
}

// filterMessage removes sensitive data from the message like the API key.
func filterMessage(message string, args ...interface{}) string {
	re := regexp.MustCompile(`(?m)"api_key"\s*:\s*"[^"]*"`)

	return re.ReplaceAllString(fmt.Sprintf(message, args...), `"api_key":"***"`)
}
