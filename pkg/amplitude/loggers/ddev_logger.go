package loggers

import (
	"fmt"
	"regexp"

	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/ddev/ddev/pkg/util"
)

func NewDdevLogger() types.Logger {
	return &ddevLogger{}
}

type ddevLogger struct {
}

func (l *ddevLogger) Debugf(message string, args ...interface{}) {
	util.Debug(filterMessage(message, args...))
}

func (l *ddevLogger) Infof(message string, args ...interface{}) {
	util.Debug(filterMessage(message, args...))
}

func (l *ddevLogger) Warnf(message string, args ...interface{}) {
	util.Debug(filterMessage(message, args...))
}

func (l *ddevLogger) Errorf(message string, args ...interface{}) {
	util.Debug(filterMessage(message, args...))
}

// filterMessage removes sensitive data from the message like the API key.
func filterMessage(message string, args ...interface{}) string {
	re := regexp.MustCompile(`(?m)"api_key"\s*:\s*"[^"]*"`)

	return re.ReplaceAllString(fmt.Sprintf(message, args...), `"api_key":"***"`)
}
