package ddevapp

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

// WriteDockerComposeYAML writes a .ddev-docker-compose-base.yaml and related to the .ddev directory.
// It then uses `docker-compose convert` to get a canonical version of the full compose file.
// It then makes a couple of fixups to the canonical version (networks and approot bind points) by
// marshaling the canonical version to YAML and then unmarshaling it back into a canonical version.
func (app *DdevApp) WriteDockerComposeYAML() error {
	var err error

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

	baseYAMLPath := app.DockerComposeYAMLPath()
	baseContentBytes := []byte(rendered)

	// If the file already exists and has the same content, don't overwrite it.
	skipBaseWrite := false
	if existingContent, err := os.ReadFile(baseYAMLPath); err == nil {
		if bytes.Equal(baseContentBytes, existingContent) {
			skipBaseWrite = true
		}
	}

	if !skipBaseWrite {
		f, err := os.Create(baseYAMLPath)
		if err != nil {
			return err
		}
		defer util.CheckClose(f)

		_, err = f.Write(baseContentBytes)
		if err != nil {
			return err
		}
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
	fullContentsBytes, err := app.ComposeYaml.MarshalYAML()
	if err != nil {
		return err
	}
	fullContentsBytes = util.EscapeDollarSign(fullContentsBytes)
	fullPath := app.DockerComposeFullRenderedYAMLPath()

	// If the file already exists and has the same content, don't overwrite it.
	skipFullWrite := false
	if existingContent, err := os.ReadFile(fullPath); err == nil {
		if bytes.Equal(fullContentsBytes, existingContent) {
			skipFullWrite = true
		}
	}

	if !skipFullWrite {
		f, err := os.Create(fullPath)
		if err != nil {
			return err
		}
		defer func() {
			err = f.Close()
			if err != nil {
				util.Warning("Error closing %s: %v", f.Name(), err)
			}
		}()

		_, err = f.Write(fullContentsBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReadDockerComposeYAML reads the rendered Docker Compose YAML file and assigns it to app.ComposeYaml
func (app *DdevApp) ReadDockerComposeYAML() error {
	content, err := fileutil.ReadFileIntoString(app.DockerComposeFullRenderedYAMLPath())
	if err != nil {
		return err
	}
	app.ComposeYaml, err = dockerutil.CreateComposeProject(content)
	if err != nil {
		return err
	}
	return nil
}

// XDdevExtension represents the x-ddev extension data in docker-compose files
type XDdevExtension struct {
	DescribeURLPort string `mapstructure:"describe-url-port"`
	DescribeInfo    string `mapstructure:"describe-info"`
	SSHShell        string `mapstructure:"ssh-shell"`
}

// GetXDdevExtension retrieves the x-ddev extension for a given service from the ComposeYaml
func (app *DdevApp) GetXDdevExtension(serviceName string) XDdevExtension {
	var xDdev XDdevExtension
	// Set defaults for web and db services
	if serviceName == "web" || serviceName == "db" {
		xDdev.SSHShell = "bash"
	}
	// And check for user overrides
	if app.ComposeYaml != nil && app.ComposeYaml.Services != nil {
		if composeService, ok := app.ComposeYaml.Services[serviceName]; ok {
			if found, err := composeService.Extensions.Get("x-ddev", &xDdev); err == nil && found {
				// Trim whitespace from all string fields
				xDdev.DescribeInfo = strings.TrimSpace(xDdev.DescribeInfo)
				xDdev.DescribeURLPort = strings.TrimSpace(xDdev.DescribeURLPort)
				xDdev.SSHShell = strings.TrimSpace(xDdev.SSHShell)
			}
		}
	}
	// Default to sh if no shell specified
	if xDdev.SSHShell == "" {
		xDdev.SSHShell = "sh"
	}
	// Append shell info to DescribeInfo if it's not the default
	hasCustomShell := false
	if serviceName == "web" || serviceName == "db" {
		hasCustomShell = xDdev.SSHShell != "bash"
	} else {
		hasCustomShell = xDdev.SSHShell != "sh"
	}
	if hasCustomShell {
		xDdev.DescribeInfo = strings.TrimSpace(fmt.Sprintf("%s\nShell: %s", xDdev.DescribeInfo, xDdev.SSHShell))
	}
	return xDdev
}

// fixupComposeYaml makes minor changes to the `docker-compose config` output
// to make sure extra services are always compatible with ddev.
func fixupComposeYaml(yamlStr string, app *DdevApp) (*composeTypes.Project, error) {
	project, err := dockerutil.CreateComposeProject(yamlStr)
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
				network.Labels = composeTypes.Labels{}
			}
			network.Labels["com.ddev.platform"] = "ddev"
		}
		project.Networks[name] = network
	}

	bindIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return project, err
	}
	// On remote Docker hosts, the Docker IP (e.g. a cloud provider's public IP)
	// is not a valid bind address on the Docker host itself, so bind to all interfaces.
	if app.BindAllInterfaces || dockerutil.IsRemoteDockerHost() {
		bindIP = "0.0.0.0"
	}

	isPodman := dockerutil.IsPodman()
	isRootless := dockerutil.IsRootless()
	isSELinux := dockerutil.IsSELinux()
	uid, gid, _ := dockerutil.GetContainerUser()
	userGroup := uid + ":" + gid

	hostDockerInternal := dockerutil.GetHostDockerInternal()

	// Ensure all services have required networks and environment variables
	for name, service := range project.Services {
		if _, ok := service.Networks[dockerutil.NetName]; !ok {
			service.Networks[dockerutil.NetName] = nil
		}
		if _, ok := service.Networks["default"]; !ok {
			service.Networks["default"] = nil
		}

		// Set up host.docker.internal based on DDEV's standard approach
		if hostDockerInternal.ExtraHost != "" {
			if service.ExtraHosts == nil {
				service.ExtraHosts = composeTypes.HostsList{}
			}
			if service.ExtraHosts["host.docker.internal"] == nil {
				service.ExtraHosts["host.docker.internal"] = []string{}
			}
			if !slices.Contains(service.ExtraHosts["host.docker.internal"], hostDockerInternal.ExtraHost) {
				service.ExtraHosts["host.docker.internal"] = append(service.ExtraHosts["host.docker.internal"], hostDockerInternal.ExtraHost)
			}
		}
		service.Environment["HOST_DOCKER_INTERNAL_IP"] = &hostDockerInternal.IPAddress

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

		// Podman: set the user namespace mode for the container
		// https://docs.podman.io/en/v4.6.1/markdown/options/userns.container.html#userns-mode
		if isPodman && isRootless && service.User == userGroup {
			service.UserNSMode = "keep-id"
		}

		if isPodman {
			if service.Links != nil || service.ExternalLinks != nil {
				util.WarningOnce("Podman does not support 'links' or 'external_links'. These options will be ignored for service '%s' in project '%s'.", name, app.Name)
			}
			service.Links = nil
			service.ExternalLinks = nil

			if service.HealthCheck != nil {
				// can't set healthcheck.start_interval as this feature requires Docker Engine v25 or later
				if service.HealthCheck.StartInterval != nil {
					service.HealthCheck.StartInterval = nil
				}
				if service.HealthCheck.Test == nil {
					util.WarningOnce("Service %s in project %s defines a healthcheck without a 'test' command. Podman may not execute it correctly.", name, app.Name)
				}
			}
		}

		if isPodman || isRootless {
			// "ping" command needs extra capability
			if !slices.Contains(service.CapAdd, "NET_RAW") {
				service.CapAdd = append(service.CapAdd, "NET_RAW")
			}
		}

		// Add SELinux labels to bind mounts when SELinux is enabled
		if isSELinux {
			for i, vol := range service.Volumes {
				if vol.Type == "bind" {
					// Initialize Bind struct if needed
					if vol.Bind == nil {
						vol.Bind = &composeTypes.ServiceVolumeBind{}
					}
					if vol.Bind.SELinux == "" {
						// Use "z" (shared) label for bind mounts
						// This allows multiple containers to share the same volume
						vol.Bind.SELinux = composeTypes.SELinuxShared
						// And set CreateHostPath to true, because SELinux doesn't work without it
						// See https://github.com/docker/compose/issues/13396
						vol.Bind.CreateHostPath = true
					}
					service.Volumes[i] = vol
				}
			}
		}

		project.Services[name] = service
	}

	return project, nil
}
