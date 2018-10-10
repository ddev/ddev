package exec

import (
	"os/exec"
	"strings"

	"github.com/drud/ddev/pkg/output"
	log "github.com/sirupsen/logrus"
)

// RunCommand runs a command on the host system.
// returns the stdout of the command and an err
func RunCommand(command string, args []string) (string, error) {
	output.UserOut.WithFields(log.Fields{
		"Command": command + " " + strings.Join(args[:], " "),
	}).Info("Running Command")

	out, err := exec.Command(
		command, args...,
	).CombinedOutput()

	output.UserOut.WithFields(log.Fields{
		"Result": string(out),
	}).Debug("Command Result")

	return string(out), err
}

// RunCommandPipe runs a command on the host system
// Returns combined output as string, and error
func RunCommandPipe(command string, args []string) (string, error) {
	output.UserOut.WithFields(log.Fields{
		"Command": command + " " + strings.Join(args[:], " "),
	}).Info("Running Command")

	cmd := exec.Command(command, args...)
	stdoutStderr, err := cmd.CombinedOutput()
	return string(stdoutStderr), err
}
