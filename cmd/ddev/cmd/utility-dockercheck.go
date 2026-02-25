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
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
)

// DebugDockercheckCmd implements the ddev utility dockercheck command
var DebugDockercheckCmd = &cobra.Command{
	Use:   "dockercheck",
	Short: "Diagnose DDEV Docker provider setup",
	Example: `ddev utility dockercheck
ddev ut dockercheck`,
	Run: func(_ *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}

		hasWarnings := false

		// Ensure docker-compose is available before getting version info
		_, dockerComposeErr := dockerutil.DownloadDockerComposeIfNeeded()

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

		case "docker-desktop":
			p, err := exec.LookPath("docker")
			if err == nil {
				out, err := exec2.RunHostCommand(bashPath, "-c", fmt.Sprintf("%s version | awk '/^Server/ {print $4}'", p))
				out = strings.Trim(out, "\r\n, ")
				if err == nil {
					util.Success("Docker Desktop version: %v", out)
				}
			}
		}

		buildxErr := dockerutil.CheckDockerBuildxVersion(dockerutil.DockerRequirements)
		if buildxErr != nil {
			util.Warning("Docker buildx version check: %v", buildxErr)
			hasWarnings = true
		} else {
			buildxVersion, _ := dockerutil.GetBuildxVersion()
			util.Success("docker buildx version %s meets requirements for minimum %s", buildxVersion, dockerutil.DockerRequirements.BuildxVersion)
		}

		dockerContextName, dockerHost, err := dockerutil.GetDockerContextNameAndHost()
		if err != nil {
			util.Warning("Could not get Docker context and host: %v", err)
			hasWarnings = true
		} else {
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
		}

		// Show TLS configuration
		dockerTLSVerify := os.Getenv("DOCKER_TLS_VERIFY")
		dockerTLS := os.Getenv("DOCKER_TLS")
		dockerCertPath := os.Getenv("DOCKER_CERT_PATH")
		if dockerTLSVerify != "" {
			util.Success("DOCKER_TLS_VERIFY=%s (TLS enabled with verification)", dockerTLSVerify)
		} else if dockerTLS != "" {
			util.Success("DOCKER_TLS=%s (TLS enabled without verification)", dockerTLS)
		} else {
			util.Success("TLS not configured (no DOCKER_TLS_VERIFY or DOCKER_TLS)")
		}
		if dockerCertPath != "" {
			util.Success("DOCKER_CERT_PATH=%s", dockerCertPath)
		}

		if dockerComposeErr != nil {
			util.Warning("Docker compose check failed: %v", dockerComposeErr)
			hasWarnings = true
		} else {
			util.Success("docker-compose: %s", versionInfo["docker-compose"])
		}

		dockerVersion, err := dockerutil.GetDockerVersion()
		if err != nil {
			util.Warning("Unable to get Docker version: %v", err)
			hasWarnings = true
		} else {
			util.Success("Docker version: %s", dockerVersion)
		}
		err = dockerutil.CheckDockerVersion(dockerutil.DockerRequirements)
		if err != nil {
			if err.Error() == "no docker" {
				util.Warning("Docker is not installed or the Docker client is not available in the $PATH")
			} else {
				util.WarningOnce("Problem with your Docker provider: %v.", err)
			}
			hasWarnings = true
		}
		dockerAPIVersion, err := dockerutil.GetDockerAPIVersion()
		if err != nil {
			util.Warning("Unable to get Docker API version: %v", err)
			hasWarnings = true
		} else {
			util.Success("Docker API version: %s", dockerAPIVersion)
		}

		uid, _, _ := dockerutil.GetContainerUser()
		_, out, err := dockerutil.RunSimpleContainer(docker.GetWebImage(), "dockercheck-runcontainer--"+util.RandString(6), []string{"ls", "/mnt/ddev-global-cache"}, []string{}, []string{}, []string{"ddev-global-cache" + ":/mnt/ddev-global-cache"}, uid, true, false, map[string]string{"com.ddev.site-name": ""}, nil, nil)
		if err != nil {
			util.Warning("Unable to run simple container: %v; output=%s", err, out)
			hasWarnings = true
		} else {
			util.Success("Able to run simple container that mounts a volume.")
		}

		_, _, err = dockerutil.RunSimpleContainer(docker.GetWebImage(), "dockercheck-curl--"+util.RandString(6), []string{"curl", "-sfLI", "https://google.com"}, []string{}, []string{}, []string{"ddev-global-cache" + ":/mnt/ddev-global-cache/bashhistory"}, uid, true, false, map[string]string{"com.ddev.site-name": ""}, nil, nil)
		if err != nil {
			util.Warning("Unable to run use internet inside container, many things will fail: %v", err)
			hasWarnings = true
		} else {
			util.Success("Able to use internet inside container.")
		}

		dockerutil.CheckAvailableSpace()

		// Test buildx with a trivial build on the host
		if buildxErr == nil {
			out, err = exec2.RunHostCommand(bashPath, "-c", fmt.Sprintf("echo 'FROM %s' | docker buildx build --quiet -f- -t ddev-buildx-test:latest . && docker rmi -f ddev-buildx-test:latest", versionconstants.UtilitiesImage))
			if err != nil {
				util.Warning("Unable to perform trivial build with buildx: %v; output=%s", err, out)
				hasWarnings = true
			} else {
				util.Success("docker buildx is working correctly (trivial build succeeded)")
			}
		} else {
			util.Warning("Skipping buildx test due to earlier buildx version check error.")
			hasWarnings = true
		}

		// Check docker auth configuration
		err = dockerutil.CheckDockerAuth()
		if err != nil {
			util.Warning("Docker authentication may have issues: %v", err)
			hasWarnings = true
		} else {
			util.Success("Docker authentication is configured correctly")
		}

		if hasWarnings {
			util.Failed("Docker provider checks completed with warnings. Please address the issues above.")
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugDockercheckCmd)
}
