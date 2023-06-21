package loggers

import (
	"log"
	"os"

	"github.com/amplitude/analytics-go/amplitude/types"
)

func NewDefaultLogger() types.Logger {
	return &defaultLogger{
		logger: log.New(os.Stderr, "amplitude-analytics - ", log.LstdFlags),
	}
}

type defaultLogger struct {
	logger *log.Logger
}

func (l *defaultLogger) Debugf(message string, args ...interface{}) {
	l.logger.Printf("Debug: "+message, args...)
}

func (l *defaultLogger) Infof(message string, args ...interface{}) {
	l.logger.Printf("Info: "+message, args...)
}

func (l *defaultLogger) Warnf(message string, args ...interface{}) {
	l.logger.Printf("Warn: "+message, args...)
}

func (l *defaultLogger) Errorf(message string, args ...interface{}) {
	l.logger.Printf("Error: "+message, args...)
}
