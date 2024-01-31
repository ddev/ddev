package exec

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/globalconfig"

	"github.com/ddev/ddev/pkg/output"
	log "github.com/sirupsen/logrus"
)

// HostCommand wraps RunCommand() to inject environment variables.
// especially DDEV_EXECUTABLE, the full path to running DDEV instance.
func HostCommand(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	ddevExecutable, _ := os.Executable()
	c.Env = append(os.Environ(),
		"DDEV_EXECUTABLE="+ddevExecutable,
	)
	return c
}

// RunCommand runs a command on the host system.
// returns the stdout of the command and an err
func RunCommand(command string, args []string) (string, error) {
	out, err := HostCommand(
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

	cmd := HostCommand(command, args...)
	stdoutStderr, err := cmd.CombinedOutput()
	return string(stdoutStderr), err
}

// RunInteractiveCommand runs a command on the host system interactively, with stdin/stdout/stderr connected
// Returns error
func RunInteractiveCommand(command string, args []string) error {
	cmd := HostCommand(command, args...)
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
	c := HostCommand(command, args...)
	c.Stdin = os.Stdin
	o, err := c.CombinedOutput()
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommand returned. output=%v err=%v", string(o), err)
	}

	return string(o), err
}

// RunHostCommandSeparateStreams executes a command on the host and returns the
// stdout and error
func RunHostCommandSeparateStreams(command string, args ...string) (string, error) {
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommandSeparateStreams: " + command + " " + strings.Join(args, " "))
	}
	c := HostCommand(command, args...)
	c.Stdin = os.Stdin
	o, err := c.Output()
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommandSeparateStreams returned. stdout=%v, err=%v", string(o), err)
	}

	return string(o), err
}

// GetHostShell gets the name of the shell running on the host
func GetHostShell() (string, error) {
	pid := "$$"
	validShells := []string{"zsh", "fish", "bash"}

	// Attempt to get the shell. Has to be in a loop for tests to pass,
	// since tests have an extra process between ddev and the shell than regular execution does.
	for pid != "1" {
		// Check if the current pid represents the shell
		comm, err := RunCommand("sh", []string{"-c", "ps -o comm= -p " + pid})
		if err != nil {
			return "", err
		}
		comm = strings.TrimSpace(comm)
		if slices.Contains(validShells, comm) {
			return comm, nil
		}
		// Try again for another round
		pid, err = RunCommand("sh", []string{"-c", "ps -o ppid= -p " + pid})
		if err != nil {
			return "", err
		}
		pid = strings.TrimSpace(pid)
	}

	return "", errors.New("couldn't determine host shell")
}
