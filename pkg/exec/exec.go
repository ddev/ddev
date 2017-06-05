package exec

import (
	"os"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// RunCommand runs a command on the host system.
func RunCommand(command string, args []string) (string, error) {
	log.WithFields(log.Fields{
		"Command": command + " " + strings.Join(args[:], " "),
	}).Info("Running Command")

	out, err := exec.Command(
		command, args...,
	).CombinedOutput()

	log.WithFields(log.Fields{
		"Result": string(out),
	}).Debug("Command Result")

	return string(out), err
}

// RunCommandPipe runs a command on the host system while piping output to stderr and stdout.
func RunCommandPipe(command string, args []string) error {
	log.WithFields(log.Fields{
		"Command": command + " " + strings.Join(args[:], " "),
	}).Info("Running Command")

	proc := exec.Command(command, args...)
	proc.Stdout = os.Stdout
	proc.Stdin = os.Stdin
	proc.Stderr = os.Stderr

	return proc.Run()
}
