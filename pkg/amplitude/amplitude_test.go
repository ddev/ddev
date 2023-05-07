package amplitude_test

import (
	"runtime"
	"testing"

	"github.com/ddev/ddev/pkg/amplitude"
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

func (t *AmplitudeSuite) TestGetUserID() {
	require := t.Require()

	require.NotEmpty(amplitude.GetUserID())
}

func (t *AmplitudeSuite) TestGetEventOptions() {
	require := t.Require()

	eventOptions := amplitude.GetEventOptions()
	require.NotEmpty(eventOptions)
	require.Equal(versionconstants.DdevVersion, eventOptions.AppVersion)
	require.Equal(runtime.GOARCH, eventOptions.Platform)
	require.Equal(runtime.GOOS, eventOptions.OSName)
}

func (t *AmplitudeSuite) TestTrackBinary() {
	require := t.Require()

	require.NotPanics(func() {
		amplitude.TrackBinary()
	})
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
