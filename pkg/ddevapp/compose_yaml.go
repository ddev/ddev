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
	"go.yaml.in/yaml/v3"
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
	fullContentsBytes, err := yaml.Marshal(app.ComposeYaml)
	if err != nil {
		return err
	}

	_, err = fullHandle.Write(fullContentsBytes)
	if err != nil {
		return err
	}

	return nil
}

// fixupComposeYaml makes minor changes to the `docker-compose config` output
// to make sure extra services are always compatible with ddev.
func fixupComposeYaml(yamlStr string, app *DdevApp) (map[string]interface{}, error) {
	envFiles, err := app.EnvFiles()
	if err != nil {
		return nil, err
	}

	cfg := composeTypes.ConfigDetails{
		WorkingDir: app.AppRoot,
		ConfigFiles: []composeTypes.ConfigFile{
			{Content: []byte(yamlStr)},
		},
	}

	project, err := composeLoader.LoadWithContext(
		context.Background(),
		cfg,
		composeLoader.WithProfiles([]string{`*`}),
	)

	if err != nil {
		return nil, err
	}

	bindIP, _ := dockerutil.GetDockerIP()
	if app.BindAllInterfaces {
		bindIP = "0.0.0.0"
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

	// Ensure all services have required networks and environment variables
	for name, service := range project.Services {
		// network_mode and networks are mutually exclusive
		if service.NetworkMode != "" {
			service.NetworkMode = ""
		}
		if service.Networks == nil {
			service.Networks = map[string]*composeTypes.ServiceNetworkConfig{}
		}
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
				if len(envMap) > 0 {
					if service.Environment == nil {
						service.Environment = map[string]*string{}
					}
					for envKey, envValue := range envMap {
						val := envValue
						service.Environment[envKey] = &val
					}
				}
			}
		}
		// Pass NO_COLOR to containers
		if !output.ColorsEnabled() {
			if serviceMap["environment"] == nil {
				serviceMap["environment"] = map[string]interface{}{}
			}
			if environmentMap, ok := serviceMap["environment"].(map[string]interface{}); ok {
				if _, exists := environmentMap["NO_COLOR"]; !exists {
					environmentMap["NO_COLOR"] = os.Getenv("NO_COLOR")
				}
			}
		}

		// Assign the host_ip for each port if it's not already set.
		// This is needed for custom-defined user ports. For example:
		// ports:
		//   - 3000:3000
		// Without this, Docker doesn't add a Docker IP, like this:
		// ports:
		//   - 127.0.0.1:3000:3000
		for port := range service.Ports {
			if service.Ports[port].HostIP == "" {
				service.Ports[port].HostIP = bindIP
			}
		}
		project.Services[name] = service
	}

	yamlBytes, err := project.MarshalYAML()
	if err != nil {
		return nil, err
	}
	yamlBytes = escapeDollarSign(yamlBytes)
	var result map[string]interface{}
	err = yaml.Unmarshal(yamlBytes, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// escapeDollarSign the same thing is done in `docker-compose config`
// See https://github.com/docker/compose/blob/361c0893a9e16d54f535cdb2e764362363d40702/cmd/compose/config.go#L405-L409
func escapeDollarSign(marshal []byte) []byte {
	dollar := []byte{'$'}
	escDollar := []byte{'$', '$'}
	return bytes.ReplaceAll(marshal, dollar, escDollar)
}
