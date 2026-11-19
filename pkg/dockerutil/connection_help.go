package dockerutil

import (
	"os"
	"runtime"

	"github.com/ddev/ddev/pkg/dockerutil/dockerhint"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/versionconstants"
)

// DockerConnectionHelp returns next steps for err, an error from connecting
// to the Docker daemon, based on the provider the Docker context points to.
func DockerConnectionHelp(err error) string {
	contextName, host, _ := GetDockerContextNameAndHost()
	return dockerhint.ConnectionHelp(dockerhint.Input{
		GOOS:          runtime.GOOS,
		WSL2:          nodeps.IsWSL2(),
		ContextName:   contextName,
		Host:          host,
		DockerHostEnv: os.Getenv("DOCKER_HOST"),
		DockerCtxEnv:  os.Getenv("DOCKER_CONTEXT"),
		Err:           err,
		DocsURL:       versionconstants.DocsURL("users/install/docker-installation/#troubleshooting-docker"),
	})
}
