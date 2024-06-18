package exec

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

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

// RunInteractiveCommandWithOutput writes to the host and
// also to the passed io.Writer
func RunInteractiveCommandWithOutput(command string, args []string, output io.Writer) error {
	cmd := HostCommand(command, args...)
	cmd.Stdin = os.Stdin

	pr, pw := io.Pipe()
	defer func() {
		_ = pr.Close()
	}()
	cmd.Stdout = pw
	cmd.Stderr = pw

	err := cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		_ = CleanAndCopy(output, pr)
		_ = pr.Close()
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Goroutine to handle signals so the script can do the right thing
	go func() {
		sig := <-sigs
		// Send the received signal to the child process
		if err := cmd.Process.Signal(sig); err != nil {
			panic(err)
		}
	}()

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

// RunHostCommandWithEnv executes a command on the host with optional
// environment variables and returns the
// combined stdout/stderr results and error
// If all of the existing environment is required, it must be
// passed in `env`, as it is not set by default
func RunHostCommandWithEnv(command string, env []string, args ...string) (string, error) {
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommandWithEnv(%v): %s %s", env, command, strings.Join(args, " "))
	}

	c := exec.Command(command, args...)
	c.Env = env
	c.Stdin = os.Stdin
	o, err := c.CombinedOutput()
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommandWithEnv returned. output=%v err=%v", string(o), err)
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

// CleanAndCopy removes control characters from output
func CleanAndCopy(dst io.Writer, src io.Reader) error {
	scanner := bufio.NewScanner(src)
	// This regex matches ANSI escape codes that are used for terminal text formatting such as color changes.
	// \x1b is the ESC character, which starts the escape sequence.
	// [^m]* matches any character that is not 'm', multiple times. 'm' is the final character in the sequence.
	// This effectively matches any escape sequence starting with ESC and ending with 'm'.
	re := regexp.MustCompile(`\x1b[^m]*m`)
	for scanner.Scan() {
		cleanString := re.ReplaceAllString(scanner.Text(), "")
		_, err := io.WriteString(dst, cleanString+"\n")
		if err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
