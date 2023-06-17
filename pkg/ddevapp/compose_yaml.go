package ddevapp

import (
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/util"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	//compose_cli "github.com/compose-spec/compose-go/cli"
	//compose_types "github.com/compose-spec/compose-go/types"
)

// WriteDockerComposeYAML writes a .ddev-docker-compose-base.yaml and related to the .ddev directory.
// It then uses `docker-compose convert` to get a canonical version of the full compose file.
// It then makes a couple of fixups to the canonical version (networks and approot bind points) by
// marshalling the canonical version to YAML and then unmarshalling it back into a canonical version.
func (app *DdevApp) WriteDockerComposeYAML() error {
	var err error

	f, err := os.Create(app.DockerComposeYAMLPath())
	if err != nil {
		return err
	}
	defer util.CheckClose(f)

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
	fullContents, _, err := dockerutil.ComposeCmd(files, "config")
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

	// Find any services that have bind-mount to AppRoot and make them relative
	// for https://youtrack.jetbrains.com/issue/WI-61976 - PhpStorm
	// This is an ugly an shortsighted approach, but otherwise we'd have to parse the yaml.
	// Note that this issue with docker-compose config was fixed in docker-compose 2.0.0RC4
	// so it's in Docker Desktop 4.1.0.
	// https://github.com/docker/compose/issues/8503#issuecomment-930969241

	for _, service := range tempMap["services"].(map[string]interface{}) {
		if service == nil {
			continue
		}
		serviceMap := service.(map[string]interface{})

		// Find any services that have bind-mount to app.AppRoot and make them relative
		if serviceMap["volumes"] != nil {
			volumes := serviceMap["volumes"].([]interface{})
			for k, volume := range volumes {
				// With docker-compose v1, the volume might not be a map, it might be
				// old-style "/Users/rfay/workspace/d9/.ddev:/mnt/ddev_config:ro"
				if volumeMap, ok := volume.(map[string]interface{}); ok {
					if volumeMap["source"] != nil {
						if volumeMap["source"].(string) == app.AppRoot {
							volumeMap["source"] = "../"
						}
					}
				} else if volumeMap, ok := volume.(string); ok {
					parts := strings.SplitN(volumeMap, ":", 2)
					if parts[0] == app.AppRoot && len(parts) >= 2 {
						volumes[k] = "../" + parts[1]
					}
				}
			}
		}
		// Make sure all services have our networks stanza
		serviceMap["networks"] = map[string]interface{}{
			"ddev_default": nil,
			"default":      nil,
		}
	}

	return tempMap, nil
}
