package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	exec2 "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/version"
	"github.com/spf13/cobra"
)

// DebugDockercheckCmd implements the ddev utility dockercheck command
var DebugDockercheckCmd = &cobra.Command{
	Use:     "dockercheck",
	Short:   "Diagnose DDEV Docker/Colima setup",
	Example: "ddev utility dockercheck",
	Run: func(_ *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}

		versionInfo := version.GetVersionInfo()
		bashPath := util.FindBashPath()
		util.Success("Docker platform: %v", versionInfo["docker-platform"])
		switch versionInfo["docker-platform"] {
		case "colima":
			p, err := exec.LookPath("colima")
			if err == nil {
				out, err := exec2.RunHostCommand(bashPath, "-c", fmt.Sprintf("%s --version | awk '{print $3}'", p))
				out = strings.Trim(out, "\r\n ")
				if err == nil {
					util.Success("Colima version: %v", out)
				}
			}
		case "lima":
			p, err := exec.LookPath("limactl")
			if err == nil {
				out, err := exec2.RunHostCommand(bashPath, "-c", fmt.Sprintf("%s --version | awk '{print $3}'", p))
				out = strings.Trim(out, "\r\n ")
				if err == nil {
					util.Success("Lima version: %v", out)
				}
			}
		case "orbstack":
			p, err := exec.LookPath("orb")
			if err == nil {
				out, err := exec2.RunHostCommand(bashPath, "-c", fmt.Sprintf("%s version | awk '/Version/ {print $2}'", p))
				out = strings.Trim(out, "\r\n ")
				if err == nil {
					util.Success("OrbStack version: %v", out)
				}
			}
		case "rancher-desktop":
			p, err := exec.LookPath("rdctl")
			if err == nil {
				out, err := exec2.RunHostCommand(bashPath, "-c", fmt.Sprintf("%s version | awk '/version/ {print $4}'", p))
				out = strings.Trim(out, "\r\n, ")
				if err == nil {
					util.Success("Rancher Desktop version: %v", out)
				}
			}

		case "docker desktop":
			dockerutil.IsDockerDesktop()
		}

		buildxVersion, err := exec2.RunHostCommand(bashPath, "-c", "docker buildx version | awk '{print $2}'")
		if err != nil {
			util.Failed("buildx is required and does not seem to be installed. Please install with 'brew install docker-buildx or see %s", buildxVersion, "https://github.com/docker/buildx#installing")
		} else {
			buildxVersion = strings.Trim(buildxVersion, "\r\n ")
			util.Success("docker buildx version %s", buildxVersion)
		}

		dockerContextName, dockerHost, err := dockerutil.GetDockerContextNameAndHost()
		if err != nil {
			util.Failed("Could not get Docker context and host: %v", err)
		}

		util.Success("Using Docker context: %s", dockerContextName)
		dockerContextName = os.Getenv("DOCKER_CONTEXT")
		if dockerContextName != "" {
			util.Success("From DOCKER_CONTEXT=%s", dockerContextName)
		}

		util.Success("Using Docker host: %s", dockerHost)
		dockerHost = os.Getenv("DOCKER_HOST")
		if dockerHost != "" {
			util.Success("From DOCKER_HOST=%s", dockerHost)
		}

		util.Success("docker-compose: %s", versionInfo["docker-compose"])

		dockerVersion, err := dockerutil.GetDockerVersion()
		if err != nil {
			util.Failed("Unable to get Docker version: %v", err)
		}
		util.Success("Docker version: %s", dockerVersion)
		err = dockerutil.CheckDockerVersion(dockerutil.DockerRequirements)
		if err != nil {
			if err.Error() == "no docker" {
				util.Failed("Docker is not installed or the Docker client is not available in the $PATH")
			} else {
				util.Warning("Problem with your Docker provider: %v.", err)
			}
		}
		dockerAPIVersion, err := dockerutil.GetDockerAPIVersion()
		if err != nil {
			util.Failed("Unable to get Docker API version: %v", err)
		}
		util.Success("Docker API version: %s", dockerAPIVersion)

		uid, _, _ := util.GetContainerUIDGid()
		_, out, err := dockerutil.RunSimpleContainer(docker.GetWebImage(), "dockercheck-runcontainer--"+util.RandString(6), []string{"ls", "/mnt/ddev-global-cache"}, []string{}, []string{}, []string{"ddev-global-cache" + ":/mnt/ddev-global-cache"}, uid, true, false, map[string]string{"com.ddev.site-name": ""}, nil, nil)
		if err != nil {
			util.Warning("Unable to run simple container: %v; output=%s", err, out)
		} else {
			util.Success("Able to run simple container that mounts a volume.")
		}

		_, _, err = dockerutil.RunSimpleContainer(docker.GetWebImage(), "dockercheck-curl--"+util.RandString(6), []string{"curl", "-sfLI", "https://google.com"}, []string{}, []string{}, []string{"ddev-global-cache" + ":/mnt/ddev-global-cache/bashhistory"}, uid, true, false, map[string]string{"com.ddev.site-name": ""}, nil, nil)
		if err != nil {
			util.Warning("Unable to run use internet inside container, many things will fail: %v", err)
		} else {
			util.Success("Able to use internet inside container.")
		}

		dockerutil.CheckAvailableSpace()
	},
}

func init() {
	DebugCmd.AddCommand(DebugDockercheckCmd)
}
