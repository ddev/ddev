package loggers_test

import (
	"bytes"
	"os"
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

	// Replace output writer, restoring the originals afterward
	origOutLevel, origErrLevel, origDebug := output.UserOut.Level, output.UserErr.Level, globalconfig.DdevDebug
	t.T().Cleanup(func() {
		output.UserOut.SetOutput(os.Stdout)
		output.UserErr.SetOutput(&output.ErrorWriter{})
		output.UserOut.Level, output.UserErr.Level, globalconfig.DdevDebug = origOutLevel, origErrLevel, origDebug
	})
	output.UserOut.SetOutput(&writer)
	output.UserErr.SetOutput(&writer)

	// Enable debug logging
	output.UserOut.Level = logrus.DebugLevel
	output.UserErr.Level = logrus.DebugLevel
	globalconfig.DdevDebug = true

	require := t.Require()

	logger := loggers.NewDdevLogger(true, true)

	require.Implements((*types.Logger)(nil), logger)

	// Messages go through output.UserErr, whose DDEV formatter prints the
	// plain message (no logrus key=value fields). Info, warning and error
	// messages may be wrapped in color codes, so check prefix and text separately.
	for _, tc := range []struct {
		log  func(string, ...any)
		text string
	}{
		{logger.Debugf, "test message 1"},
		{logger.Errorf, "test message 2"},
		{logger.Infof, "test message 3"},
		{logger.Warnf, "test message 4"},
	} {
		writer.Reset()
		tc.log(tc.text)
		require.Contains(writer.String(), "AMPLITUDE: ")
		require.Contains(writer.String(), tc.text)
	}

	// The API key is redacted
	writer.Reset()
	logger.Debugf(`payload {"api_key": "%s", "events": []}`, "secret-key")
	require.Contains(writer.String(), `"api_key":"***"`)
	require.NotContains(writer.String(), "secret-key")

	// Nothing is logged when neither debug nor verbose is set
	quiet := loggers.NewDdevLogger(false, false)
	writer.Reset()
	quiet.Debugf("quiet 1")
	quiet.Infof("quiet 2")
	quiet.Warnf("quiet 3")
	quiet.Errorf("quiet 4")
	require.Empty(writer.String())
}
