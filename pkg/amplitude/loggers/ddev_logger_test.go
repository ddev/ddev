package loggers_test

import (
	"bytes"
	"testing"

	"github.com/amplitude/analytics-go/amplitude/types"
	"github.com/ddev/ddev/pkg/amplitude/loggers"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

func TestDdevLogger(t *testing.T) {
	suite.Run(t, new(DdevLoggerSuite))
}

type DdevLoggerSuite struct {
	suite.Suite
}

func (t *DdevLoggerSuite) TestLogger() {
	var writer bytes.Buffer

	// Replace output writer
	output.UserOut.SetOutput(&writer)
	output.UserErr.SetOutput(&writer)

	// Enable debug logging
	output.UserOut.Level = logrus.DebugLevel
	output.UserErr.Level = logrus.DebugLevel
	globalconfig.DdevDebug = true

	require := t.Require()

	logger := loggers.NewDdevLogger(true, true)

	require.Implements((*types.Logger)(nil), logger)

	writer.Reset()
	logger.Debugf("test message 1")
	require.Contains(writer.String(), "msg=\"test message 1\"")

	writer.Reset()
	logger.Errorf("test message 2")
	require.Contains(writer.String(), "msg=\"test message 2\"")

	writer.Reset()
	logger.Infof("test message 3")
	require.Contains(writer.String(), "msg=\"test message 3\"")

	writer.Reset()
	logger.Warnf("test message 4")
	require.Contains(writer.String(), "msg=\"test message 4\"")
}
