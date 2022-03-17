package exec

import (
	"github.com/drud/ddev/pkg/globalconfig"
	"os"
	"os/exec"
	"strings"

	"github.com/drud/ddev/pkg/output"
	log "github.com/sirupsen/logrus"
)

// RunCommand runs a command on the host system.
// returns the stdout of the command and an err
func RunCommand(command string, args []string) (string, error) {
	out, err := exec.Command(
		command, args...,
	).CombinedOutput()

	output.UserOut.WithFields(log.Fields{
		"Result": string(out),
	}).Debug("Command ")

	return string(out), err
}

// RunCommandPipe runs a command on the host system
// Returns combined output as string, and error
func RunCommandPipe(command string, args []string) (string, error) {
	output.UserOut.WithFields(log.Fields{
		"Command": command + " " + strings.Join(args[:], " "),
	}).Info("Running ")

	cmd := exec.Command(command, args...)
	stdoutStderr, err := cmd.CombinedOutput()
	return string(stdoutStderr), err
}

// RunInteractiveCommand runs a command on the host system interactively, with stdin/stdout/stderr connected
// Returns error
func RunInteractiveCommand(command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	return err
}

// RunHostCommand executes a command on the host and returns the
// combined stdout/stderr results and error
func RunHostCommand(command string, args ...string) (string, error) {
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommand: " + command + " " + strings.Join(args, " "))
	}
	o, err := exec.Command(command, args...).CombinedOutput()
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommand returned. output=%v err=%v", string(o), err)
	}

	return string(o), err
}
