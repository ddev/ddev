package ddevapp

import (
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/util"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	//compose_cli "github.com/compose-spec/compose-go/cli"
	//compose_types "github.com/compose-spec/compose-go/types"
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
	tempMap := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(yamlStr), &tempMap)
	if err != nil {
		return nil, err
	}
	envFiles, err := app.EnvFiles()
	if err != nil {
		return nil, err
	}

	// Ensure that some important network properties are not overridden by users
	for name, network := range tempMap["networks"].(map[string]interface{}) {
		if network == nil {
			continue
		}
		networkMap := network.(map[string]interface{})
		// Default networks don't allow to override these properties
		if name == "ddev_default" {
			networkMap["name"] = dockerutil.NetName
			networkMap["external"] = true
		}
		if name == "default" {
			networkMap["name"] = app.GetDefaultNetworkName()
			// If "external" was added by user, remove it
			delete(networkMap, "external")
		}
		// Add labels that are used to clean up internal networks when the project is stopped
		if external, ok := networkMap["external"].(bool); !ok || !external {
			labels, ok := networkMap["labels"].(map[string]interface{})
			if !ok {
				labels = make(map[string]interface{})
				networkMap["labels"] = labels
			}
			labels["com.ddev.platform"] = "ddev"
		}
	}

	// Make sure that all services have the `ddev_default` and `default` networks
	for name, service := range tempMap["services"].(map[string]interface{}) {
		if service == nil {
			continue
		}
		serviceMap := service.(map[string]interface{})

		// Make sure all services have our networks stanza
		networks, ok := serviceMap["networks"].(map[string]interface{})
		if !ok {
			networks = make(map[string]interface{})
		}
		// Add default networks if they don't exist
		if _, exists := networks["ddev_default"]; !exists {
			networks["ddev_default"] = nil
		}
		if _, exists := networks["default"]; !exists {
			networks["default"] = nil
		}
		// Update the serviceMap with the networks
		serviceMap["networks"] = networks

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
					if serviceMap["environment"] == nil {
						serviceMap["environment"] = map[string]interface{}{}
					}
					if environmentMap, ok := serviceMap["environment"].(map[string]interface{}); ok {
						for envKey, envValue := range envMap {
							// Escape $ characters in environment variables
							// The same thing is done in `docker-compose config`
							// See https://github.com/docker/compose/blob/361c0893a9e16d54f535cdb2e764362363d40702/cmd/compose/config.go#L405-L409
							environmentMap[envKey] = strings.ReplaceAll(envValue, `$`, `$$`)
						}
					}
				}
			}
		}
	}

	return tempMap, nil
}
