package amplitude_test

import (
	"testing"

	"github.com/ddev/ddev/pkg/amplitude"
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
