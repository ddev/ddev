package ddevapp

import (
	"fmt"
	"strings"

	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
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
	defer util.TimeTrack()()

	// Initialization is currently done before via init() func somewhere while
	// creating the ddevapp. This should be cleaned up.
	// TODO remove once clean up has done.
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

	builder := ampli.Project.Builder().
		Containers(services).
		ContainersOmitted(containersOmitted).
		FailOnHookFail(app.FailOnHookFail || app.FailOnHookFailGlobal).
		Id(app.ProtectedID()).
		MutagenEnabled(app.IsMutagenEnabled()).
		NfsMountEnabled(app.IsNFSMountEnabled()).
		NodejsVersion(app.NodeJSVersion).
		PhpVersion(app.GetPhpVersion()).
		ProjectType(app.GetType()).
		RouterDisabled(IsRouterDisabled(app)).
		TraefikEnabled(globalconfig.DdevGlobalConfig.UseTraefik).
		WebserverType(app.GetWebserverType())

	if !nodeps.ArrayContainsString(containersOmitted, "db") {
		builder.
			DatabaseType(app.Database.Type).
			DatabaseVersion(app.Database.Version)
	}

	ampli.Instance.Project(amplitude.GetUserID(), builder.Build())

	/*
		appDesc["xdebug_enabled"] = app.XdebugEnabled
		appDesc["webimg"] = app.WebImage
		appDesc["dbimg"] = app.GetDBImage()
		appDesc["services"] = map[string]map[string]string{}

		containers, err := dockerutil.GetAppContainers(app.Name)
		if err != nil {
			return nil, err
		}
		services := appDesc["services"].(map[string]map[string]string)
		for _, k := range containers {
			serviceName := strings.TrimPrefix(k.Names[0], "/")
			shortName := strings.Replace(serviceName, fmt.Sprintf("ddev-%s-", app.Name), "", 1)

			c, err := dockerutil.InspectContainer(serviceName)
			if err != nil || c == nil {
				util.Warning("Could not get container info for %s", serviceName)
				continue
			}
			fullName := strings.TrimPrefix(serviceName, "/")
			services[shortName] = map[string]string{}
			services[shortName]["status"] = c.State.Status
			services[shortName]["full_name"] = fullName
			services[shortName]["image"] = strings.TrimSuffix(c.Config.Image, fmt.Sprintf("-%s-built", app.Name))
			services[shortName]["short_name"] = shortName
			var ports []string
			for pk := range c.Config.ExposedPorts {
				ports = append(ports, pk.Port())
			}
			services[shortName]["exposed_ports"] = strings.Join(ports, ",")
			var hostPorts []string
			for _, pv := range k.Ports {
				if pv.PublicPort != 0 {
					hostPorts = append(hostPorts, strconv.FormatInt(pv.PublicPort, 10))
				}
			}
			services[shortName]["host_ports"] = strings.Join(hostPorts, ",")

			// Extract HTTP_EXPOSE and HTTPS_EXPOSE for additional info
			if !IsRouterDisabled(app) {
				for _, e := range c.Config.Env {
					split := strings.SplitN(e, "=", 2)
					envName := split[0]
					if len(split) == 2 && (envName == "HTTP_EXPOSE" || envName == "HTTPS_EXPOSE") {
						envVal := split[1]

						envValStr := fmt.Sprintf("%s", envVal)
						portSpecs := strings.Split(envValStr, ",")
						// There might be more than one exposed UI port, but this only handles the first listed,
						// most often there's only one.
						if len(portSpecs) > 0 {
							// HTTPS portSpecs typically look like <exposed>:<containerPort>, for example - HTTPS_EXPOSE=1359:1358
							ports := strings.Split(portSpecs[0], ":")
							//services[shortName][envName.(string)] = ports[0]
						}
					}
				}
			}
		}
	*/
}
