package loggers

import (
	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/ddev/ddev/pkg/util"
)

func NewDdevLogger() types.Logger {
	return &ddevLogger{}
}

type ddevLogger struct {
}

func (l *ddevLogger) Debugf(message string, args ...interface{}) {
	util.Debug(message, args...)
}

func (l *ddevLogger) Infof(message string, args ...interface{}) {
	util.Success(message, args...)
}

func (l *ddevLogger) Warnf(message string, args ...interface{}) {
	util.Warning(message, args...)
}

func (l *ddevLogger) Errorf(message string, args ...interface{}) {
	util.Error(message, args...)
}
