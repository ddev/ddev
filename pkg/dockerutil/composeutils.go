package dockerutil

import (
	"context"
	"os"
	"regexp"
	"strings"

	composeLoader "github.com/compose-spec/compose-go/v2/loader"
	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/mattn/go-isatty"
)

// CreateComposeProject creates a compose project from a string
func CreateComposeProject(yamlStr string) (*composeTypes.Project, error) {
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
	// Initialize Networks, Services, and Volumes to empty maps if nil
	if project.Networks == nil {
		project.Networks = composeTypes.Networks{}
	}
	if project.Services == nil {
		project.Services = composeTypes.Services{}
	}
	if project.Volumes == nil {
		project.Volumes = composeTypes.Volumes{}
	}
	// Ensure nested fields like Labels, Networks, and Environment are initialized
	for name, network := range project.Networks {
		if network.Labels == nil {
			network.Labels = composeTypes.Labels{}
		}
		project.Networks[name] = network
	}
	for name, service := range project.Services {
		if service.Networks == nil {
			service.Networks = map[string]*composeTypes.ServiceNetworkConfig{}
		}
		if service.Environment == nil {
			service.Environment = composeTypes.MappingWithEquals{}
		}
		project.Services[name] = service
	}
	return project, nil
}

// PullImages pulls images in parallel if they don't exist locally
// If pullAlways is true, it will always pull
// Otherwise, it will only pull if the image doesn't exist
func PullImages(images []string, pullAlways bool) error {
	if len(images) == 0 {
		return nil
	}

	composeYamlPull, err := CreateComposeProject("name: compose-yaml-pull")
	if err != nil {
		return err
	}

	for _, image := range images {
		if image == "" {
			continue
		}
		if !pullAlways {
			if imageExists, _ := ImageExistsLocally(image); imageExists {
				continue
			}
		}
		service := sanitizeServiceName(image)
		if _, exists := composeYamlPull.Services[service]; exists {
			continue
		}
		composeYamlPull.Services[service] = composeTypes.ServiceConfig{
			Image: image,
		}
		util.Debug(`Pulling image for %s ("%s" service)`, image, service)
	}

	if !output.JSONOutput && isatty.IsTerminal(os.Stdin.Fd()) {
		err = ComposeWithStreams(&ComposeCmdOpts{
			ComposeYaml: composeYamlPull,
			Action:      []string{"pull"},
		}, nil, os.Stdout, os.Stderr)
	} else {
		_, _, err = ComposeCmd(&ComposeCmdOpts{
			ComposeYaml: composeYamlPull,
			Action:      []string{"pull"},
		})
	}

	return err
}

// Pull pulls image if it doesn't exist locally
func Pull(image string) error {
	return PullImages([]string{image}, false)
}

// sanitizeServiceName sanitizes a string to be a valid Docker Compose service name
// by replacing any characters that don't match [a-zA-Z0-9._-] with hyphens
// See https://github.com/compose-spec/compose-go/blob/main/schema/compose-spec.json for allowed pattern
func sanitizeServiceName(input string) string {
	if input == "" {
		return ""
	}

	invalidChars := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	sanitized := invalidChars.ReplaceAllString(input, "-")

	multipleHyphens := regexp.MustCompile(`-+`)
	sanitized = multipleHyphens.ReplaceAllString(sanitized, "-")

	sanitized = strings.Trim(sanitized, "-")

	return sanitized
}
