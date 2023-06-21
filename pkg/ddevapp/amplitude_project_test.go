package ddevapp_test

import (
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/stretchr/testify/suite"
)

func TestAmplitude(t *testing.T) {
	suite.Run(t, new(AmplitudeSuite))
}

type AmplitudeSuite struct {
	suite.Suite
}

func (t *AmplitudeSuite) TestTrackProject() {
	app, err := ddevapp.NewApp("", false)
	require := t.Require()

	require.NoError(err)

	require.NotPanics(func() {
		app.TrackProject()
	})
}

func (t *AmplitudeSuite) TestProtectedID() {
	app, err := ddevapp.NewApp("", false)
	require := t.Require()

	require.NoError(err)
	require.NotEmpty(app.ProtectedID())
}
