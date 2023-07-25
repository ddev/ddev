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

// DebugDockercheckCmd implements the ddev debug dockercheck command
var DebugDockercheckCmd = &cobra.Command{
	Use:     "dockercheck",
	Short:   "Diagnose DDEV docker/colima setup",
	Example: "ddev debug dockercheck",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}

		versionInfo := version.GetVersionInfo()
		util.Success("Docker platform: %v", versionInfo["docker-platform"])
		switch versionInfo["docker-platform"] {
		case "colima":
			p, err := exec.LookPath("colima")
			if err == nil {
				bashPath := util.FindBashPath()
				out, err := exec2.RunHostCommand(bashPath, "-c", fmt.Sprintf("%s version | awk '/colima version/ {print $3}'", p))
				out = strings.Trim(out, "\r\n ")
				if err == nil {
					util.Success("Colima version: %v", out)
				}
			}

		case "docker desktop":
			dockerutil.IsDockerDesktop()
		}
		util.Success("Using docker context: %s (%s)", dockerutil.DockerContext, dockerutil.DockerHost)
		util.Success("docker-compose: %s", versionInfo["docker-compose"])

		dockerHost := os.Getenv("DOCKER_HOST")
		if dockerHost != "" {
			util.Success("Using DOCKER_HOST=%s", dockerHost)
		}
		dockerContext := os.Getenv("DOCKER_CONTEXT")
		if dockerContext != "" {
			util.Success("Using DOCKER_CONTEXT=%s", dockerContext)
		}

		dockerVersion, err := dockerutil.GetDockerVersion()
		if err != nil {
			util.Failed("Unable to get docker version: %v", err)
		}
		util.Success("Docker version: %s", dockerVersion)
		err = dockerutil.CheckDockerVersion(dockerutil.DockerVersionConstraint)
		if err != nil {
			if err.Error() == "no docker" {
				util.Failed("Docker is not installed or the docker client is not available in the $PATH")
			} else {
				util.Warning("The docker version currently installed does not seem to meet ddev's requirements: %v", err)
			}
		}

		client := dockerutil.GetDockerClient()
		if client == nil {
			util.Failed("Unable to get docker client")
		}

		uid, _, _ := util.GetContainerUIDGid()
		_, out, err := dockerutil.RunSimpleContainer(docker.GetWebImage(), "dockercheck-runcontainer--"+util.RandString(6), []string{"ls", "/mnt/ddev-global-cache"}, []string{}, []string{}, []string{"ddev-global-cache" + ":/mnt/ddev-global-cache"}, uid, true, false, map[string]string{"com.ddev.site-name": ""}, nil)
		if err != nil {
			util.Warning("Unable to run simple container: %v; output=%s", err, out)
		} else {
			util.Success("Able to run simple container that mounts a volume.")
		}

		_, _, err = dockerutil.RunSimpleContainer(docker.GetWebImage(), "dockercheck-curl--"+util.RandString(6), []string{"curl", "-sfLI", "https://google.com"}, []string{}, []string{}, []string{"ddev-global-cache" + ":/mnt/ddev-global-cache/bashhistory"}, uid, true, false, map[string]string{"com.ddev.site-name": ""}, nil)
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
