package ddevapp

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	composeLoader "github.com/compose-spec/compose-go/v2/loader"
	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

// WriteDockerComposeYAML writes a .ddev-docker-compose-base.yaml and related to the .ddev directory.
// It then uses `docker-compose convert` to get a canonical version of the full compose file.
// It then makes a couple of fixups to the canonical version (networks and approot bind points) by
// marshaling the canonical version to YAML and then unmarshaling it back into a canonical version.
func (app *DdevApp) WriteDockerComposeYAML() error {
	var err error

	f, err := os.Create(app.DockerComposeYAMLPath())
	if err != nil {
		return err
	}
	defer util.CheckClose(f)

	// Create a host working_dir for the web service beforehand.
	// Otherwise, Docker will create it as root user (when Mutagen is disabled).
	// This problem (particularly for Docker volumes) is described in
	// https://github.com/moby/moby/issues/2259
	hostWorkingDir := app.GetHostWorkingDir("web", "")
	if hostWorkingDir != "" {
		_ = os.MkdirAll(hostWorkingDir, 0755)
	}

	rendered, err := app.RenderComposeYAML()
	if err != nil {
		return err
	}
	_, err = f.WriteString(rendered)
	if err != nil {
		return err
	}

	files, err := app.ComposeFiles()
	if err != nil {
		return err
	}
	envFiles, err := app.EnvFiles()
	if err != nil {
		return err
	}
	var action []string
	for _, envFile := range envFiles {
		action = append(action, "--env-file", envFile)
	}
	fullContents, _, err := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: files,
		Profiles:     []string{`*`},
		Action:       append(action, "config"),
	})
	if err != nil {
		return err
	}

	app.ComposeYaml, err = fixupComposeYaml(fullContents, app)
	if err != nil {
		return err
	}
	fullHandle, err := os.Create(app.DockerComposeFullRenderedYAMLPath())
	if err != nil {
		return err
	}
	defer func() {
		err = fullHandle.Close()
		if err != nil {
			util.Warning("Error closing %s: %v", fullHandle.Name(), err)
		}
	}()
	fullContentsBytes, err := app.ComposeYaml.MarshalYAML()
	if err != nil {
		return err
	}
	fullContentsBytes = escapeDollarSign(fullContentsBytes)

	_, err = fullHandle.Write(fullContentsBytes)
	if err != nil {
		return err
	}

	return nil
}

// createComposeProject creates a compose project from a string
func createComposeProject(yamlStr string) (*composeTypes.Project, error) {
	project, err := composeLoader.LoadWithContext(
		context.Background(),
		composeTypes.ConfigDetails{
			ConfigFiles: []composeTypes.ConfigFile{
				{Content: []byte(yamlStr)},
			},
		},
		composeLoader.WithProfiles([]string{`*`}),
	)
	if err != nil {
		return project, err
	}
	if project.Networks == nil {
		project.Networks = make(map[string]composeTypes.NetworkConfig)
	}
	if project.Services == nil {
		project.Services = make(map[string]composeTypes.ServiceConfig)
	}
	if project.Volumes == nil {
		project.Volumes = make(map[string]composeTypes.VolumeConfig)
	}
	for name, service := range project.Services {
		if service.Networks == nil {
			service.Networks = map[string]*composeTypes.ServiceNetworkConfig{}
		}
		if service.Environment == nil {
			service.Environment = map[string]*string{}
		}
		project.Services[name] = service
	}
	return project, nil
}

// fixupComposeYaml makes minor changes to the `docker-compose config` output
// to make sure extra services are always compatible with ddev.
func fixupComposeYaml(yamlStr string, app *DdevApp) (*composeTypes.Project, error) {
	project, err := createComposeProject(yamlStr)
	if err != nil {
		return project, err
	}

	envFiles, err := app.EnvFiles()
	if err != nil {
		return project, err
	}

	// Ensure that some important network properties are not overridden by users
	if _, ok := project.Networks[dockerutil.NetName]; !ok {
		project.Networks[dockerutil.NetName] = composeTypes.NetworkConfig{}
	}
	if _, ok := project.Networks["default"]; !ok {
		project.Networks["default"] = composeTypes.NetworkConfig{}
	}
	for name, network := range project.Networks {
		if name == dockerutil.NetName {
			network.Name = dockerutil.NetName
			network.External = true
		} else if name == "default" {
			network.Name = app.GetDefaultNetworkName()
			network.External = false
		}
		if !network.External {
			if network.Labels == nil {
				network.Labels = map[string]string{}
			}
			network.Labels["com.ddev.platform"] = "ddev"
		}
		project.Networks[name] = network
	}

	bindIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return project, err
	}
	if app.BindAllInterfaces {
		bindIP = "0.0.0.0"
	}

	// Ensure all services have required networks and environment variables
	for name, service := range project.Services {
		if _, ok := service.Networks[dockerutil.NetName]; !ok {
			service.Networks[dockerutil.NetName] = nil
		}
		if _, ok := service.Networks["default"]; !ok {
			service.Networks["default"] = nil
		}

		// Add environment variables from .env files to services
		for _, envFile := range envFiles {
			filename := filepath.Base(envFile)
			// Variables from .ddev/.env should be available in all containers,
			// and variables from .ddev/.env.* should only be available in a specific container.
			if filename == ".env" || filename == ".env."+name {
				envMap, _, err := ReadProjectEnvFile(envFile)
				if err != nil && !os.IsNotExist(err) {
					util.Failed("Unable to read %s file: %v", envFile, err)
				}
				for envKey, envValue := range envMap {
					val := envValue
					service.Environment[envKey] = &val
				}
			}
		}
		// Pass NO_COLOR to containers
		if !output.ColorsEnabled() && service.Environment["NO_COLOR"] == nil {
			noColor := os.Getenv("NO_COLOR")
			service.Environment["NO_COLOR"] = &noColor
		}

		// Assign the host_ip for each port if it's not already set.
		// This is needed for custom-defined user ports. For example:
		// ports:
		//   - 3000:3000
		// Without this, Docker doesn't add a Docker IP, like this:
		// ports:
		//   - 127.0.0.1:3000:3000
		for i, port := range service.Ports {
			if port.HostIP == "" {
				port.HostIP = bindIP
			}
			service.Ports[i] = port
		}
		project.Services[name] = service
	}

	return project, nil
}

// escapeDollarSign the same thing is done in `docker-compose config`
// See https://github.com/docker/compose/blob/361c0893a9e16d54f535cdb2e764362363d40702/cmd/compose/config.go#L405-L409
func escapeDollarSign(marshal []byte) []byte {
	dollar := []byte{'$'}
	escDollar := []byte{'$', '$'}
	return bytes.ReplaceAll(marshal, dollar, escDollar)
}
