package ddevapp

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/third_party/ampli"
	"github.com/denisbrodbeck/machineid"
)

// TrackProject collects and tracks information about the project for instrumentation.
func (app *DdevApp) TrackProject(eventOptions ...ampli.EventOptions) {
	runTime := util.TimeTrack()
	defer runTime()

	builder := ampli.Project.Builder().
		Id(app.ProtectedID())

	//builder.

	ampli.Instance.Project("", builder.Build(), eventOptions...)
}

func (app *DdevApp) ProtectedID() string {
	appID, _ := machineid.ProtectedID("ddev" + app.Name)
	return appID
}
