package exec

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
)

// CmdOption is a function type for configuring exec.Cmd
type CmdOption func(*exec.Cmd)

// WithStdin sets the stdin for the host command
func WithStdin(stdin io.Reader) CmdOption {
	return func(cmd *exec.Cmd) {
		if globalconfig.DdevVerbose {
			output.UserOut.Printf("WithStdin: setting custom stdin")
		}
		cmd.Stdin = stdin
	}
}

// WithEnv sets the environment variables for the host command
func WithEnv(env []string) CmdOption {
	return func(cmd *exec.Cmd) {
		if globalconfig.DdevVerbose {
			output.UserOut.Printf("WithEnv: setting env vars %v", env)
		}
		if cmd.Env == nil {
			cmd.Env = os.Environ()
		}
		cmd.Env = append(cmd.Env, env...)
	}
}

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

	output.UserOut.WithFields(output.Fields{
		"Result": string(out),
	}).Debug("Command ")

	return string(out), err
}

// RunCommandPipe runs a command on the host system
// Returns combined output as string, and error
func RunCommandPipe(command string, args []string) (string, error) {
	output.UserOut.WithFields(output.Fields{
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

// RunInteractiveCommandWithCapture runs a command interactively like
// RunInteractiveCommand(), but also returns a copy of what it printed, so a
// failure can be reported with its output attached. stdout and stderr are
// combined and share one writer, so os/exec copies them from a single
// goroutine and the capture stays race-free.
func RunInteractiveCommandWithCapture(command string, args []string) (string, error) {
	var captured bytes.Buffer
	cmd := HostCommand(command, args...)
	cmd.Stdin = os.Stdin
	// Byte-for-byte tee, unlike RunInteractiveCommandWithOutput(), so progress
	// output and colors reach the terminal unchanged.
	tee := io.MultiWriter(os.Stdout, &captured)
	cmd.Stdout = tee
	cmd.Stderr = tee
	err := cmd.Run()
	return captured.String(), err
}

// RunInteractiveCommandWithOutput connects stdin and writes the command's
// combined output to the passed io.Writer, line by line with ANSI color
// codes removed. Nothing goes to the terminal unless out writes there.
func RunInteractiveCommandWithOutput(command string, args []string, out io.Writer) error {
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
		_ = CleanAndCopy(out, pr)
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
		output.UserOut.Printf("RunHostCommand: %s %v", command, strings.Join(args, " "))
	}
	c := HostCommand(command, args...)
	c.Stdin = os.Stdin
	o, err := c.CombinedOutput()
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommand returned output=%v err=%v", string(bytes.TrimSpace(o)), err)
	}

	return string(o), err
}

// RunHostCommandWithOptions executes a command on the host with configurable options
// and returns the combined stdout/stderr results and error
func RunHostCommandWithOptions(command string, options []CmdOption, args ...string) (string, error) {
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommandWithOptions: %s %s", command, strings.Join(args, " "))
	}
	c := HostCommand(command, args...)

	// Apply all options
	for _, option := range options {
		option(c)
	}
	// Default to os.Stdin if no stdin was set
	if c.Stdin == nil {
		c.Stdin = os.Stdin
	}

	o, err := c.CombinedOutput()
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommandWithOptions returned output=%v err=%v", string(o), err)
	}

	return string(o), err
}

// RunHostCommandSeparateStreams executes a command on the host and returns the
// stdout and error
func RunHostCommandSeparateStreams(command string, args ...string) (string, error) {
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommandSeparateStreams: %s %v", command, strings.Join(args, " "))
	}
	c := HostCommand(command, args...)
	c.Stdin = os.Stdin
	o, err := c.Output()
	if globalconfig.DdevVerbose {
		output.UserOut.Printf("RunHostCommandSeparateStreams returned output=%v, err=%v", string(o), err)
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
