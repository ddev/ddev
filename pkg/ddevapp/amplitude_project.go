package ddevapp

import (
	"fmt"
	"strings"

	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/third_party/ampli"
	"github.com/denisbrodbeck/machineid"
)

// ProtectedID returns the unique hash value for the project.
func (app *DdevApp) ProtectedID() string {
	appID, _ := machineid.ProtectedID("ddev" + app.Name)
	return appID
}

// TrackProject collects and tracks information about the project for instrumentation.
func (app *DdevApp) TrackProject() {
	// defer util.TimeTrack()()

	// Initialization is currently done before via init() func somewhere while
	// creating the ddevapp. This should be cleaned up.
	// TODO: Remove once clean up has done.
	amplitude.InitAmplitude()

	// Early exit if instrumentation is disabled.
	if ampli.Instance.Disabled {
		return
	}

	containersOmitted := app.GetOmittedContainers()

	var services []string
	containers, err := dockerutil.GetAppContainers(app.Name)
	if err == nil {
		for _, k := range containers {
			serviceName := strings.TrimPrefix(k.Names[0], "/")
			shortName := strings.Replace(serviceName, fmt.Sprintf("ddev-%s-", app.Name), "", 1)
			services = append(services, shortName)
		}
	}

	//builder := ampli.Project.Builder().Id(app.ProtectedID()).
	//
	//builder := ampli.Project.Builder().Id(app.ProtectedID()).PerformanceMode().PhpVersion().ProjectType().WebserverType().DdevVersionConstraint().NoProjectMount().WebimageExtraPackages().DbImageExtraPackages().DatabaseType().DatabaseVersion().CorepackEnable()a
	builder := ampli.Project.Builder().
		Id(app.ProtectedID()).
		PerformanceMode(app.GetPerformanceMode()).
		PhpVersion(app.GetPhpVersion()).
		ProjectType(app.GetType()).
		WebserverType(app.GetWebserverType()).
		AddOns(GetInstalledAddonNames(app)).
		Containers(services).
		ContainersOmitted(containersOmitted).
		FailOnHookFail(app.FailOnHookFail || app.FailOnHookFailGlobal).
		NodejsVersion(app.NodeJSVersion).
		RouterDisabled(IsRouterDisabled(app)).
		WebimageExtraPackages(app.WebImageExtraPackages).
		DbImageExtraPackages(app.DBImageExtraPackages).
		BindAllInterfaces(app.BindAllInterfaces).
		CorepackEnable(app.CorepackEnable).
		DdevVersionConstraint(app.DdevVersionConstraint).
		DisableSettingsManagement(app.DisableSettingsManagement).
		NoProjectMount(app.NoProjectMount)

	if !nodeps.ArrayContainsString(containersOmitted, "db") {
		builder.
			DatabaseType(app.Database.Type).
			DatabaseVersion(app.Database.Version)
	}

	if !IsRouterDisabled(app) {
		builder.Router(globalconfig.DdevGlobalConfig.Router)
	}

	ampli.Instance.Project("", builder.Build(), amplitude.GetEventOptions())
}
