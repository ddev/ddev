package amplitude_test

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

func TestAmplitude(t *testing.T) {
	suite.Run(t, new(AmplitudeSuite))
}

type AmplitudeSuite struct {
	suite.Suite
}

func (t *AmplitudeSuite) TestGetDeviceID() {
	require := t.Require()

	require.NotEmpty(amplitude.GetDeviceID())
}

func (t *AmplitudeSuite) TestGetEventOptions() {
	require := t.Require()

	require.Equal(versionconstants.DdevVersion, amplitude.GetEventOptions().AppVersion)
	require.Equal(amplitude.GetDeviceID(), amplitude.GetEventOptions().DeviceID)
	require.Equal(os.Getenv("LANG"), amplitude.GetEventOptions().Language)
	require.Equal(runtime.GOOS, amplitude.GetEventOptions().OSName)
	require.Equal(runtime.GOARCH, amplitude.GetEventOptions().Platform)
	require.Equal("ddev cli", amplitude.GetEventOptions().ProductID)
	require.LessOrEqual(amplitude.GetEventOptions().Time, time.Now().UnixMilli())
	require.Equal(globalconfig.DdevGlobalConfig.InstrumentationUser, amplitude.GetEventOptions().UserID)
}

func (t *AmplitudeSuite) TestTrackCommand() {
	require := t.Require()

	cmd := cobra.Command{}
	args := []string{"arg1", "arg2"}

	require.NotPanics(func() {
		amplitude.TrackCommand(&cmd, args)
	})
}

func (t *AmplitudeSuite) TestFlush() {
	require := t.Require()

	require.NotPanics(func() {
		amplitude.Flush()
	})
}

func (t *AmplitudeSuite) TestFlushForce() {
	require := t.Require()

	require.NotPanics(func() {
		amplitude.FlushForce()
	})
}

func (t *AmplitudeSuite) TestClean() {
	require := t.Require()

	require.NotPanics(func() {
		amplitude.Clean()
	})
}

func (t *AmplitudeSuite) TestCheckSetUp() {
	require := t.Require()

	require.NotPanics(func() {
		amplitude.CheckSetUp()
	})
}
