package drudapi

import (
	"os"
	"os/exec"
)

// DockerCompose serves as a wrapper to docker-compose
func DockerCompose(arg ...string) error {
	proc := exec.Command("docker-compose", arg...)
	proc.Stdout = os.Stdout
	proc.Stdin = os.Stdin
	proc.Stderr = os.Stderr

	return proc.Run()
}
