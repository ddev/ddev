package dockerutil

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ddev/ddev/pkg/interrupt"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	"github.com/mattn/go-isatty"
)

// An exec that reads the terminal runs in a child ddev process. docker/cli's
// RunExec returns without stopping its stdin reader (beginInputStream in
// https://github.com/docker/cli/blob/v29.9.2/cli/command/container/hijack.go#L167-L173),
// which then drops the next line typed at the terminal. ddev can't stop it
// in-process: docker/cli takes a terminal only as an *os.File, and Go can't
// interrupt a blocking read on one. The read ends when the child exits.
// If docker/cli ever stops the reader, HandleTerminalExec, its call in main()
// and ExecInChild can go, and ExecWithStreams can always run in-process.
const (
	terminalExecArg         = "__terminal-exec"
	terminalExecOptionsFile = "options.json"
	terminalExecResultFile  = "result.json"
)

// canExecInChild is set only in the ddev binary. Other binaries, such as
// tests, run every exec in-process, which leaves the reader, and on Linux
// closes os.Stdin after a TTY exec.
var canExecInChild bool

type terminalExec struct {
	ProjectName string
	Options     api.RunOptions
}

// terminalExecResult is what compose's Exec returned in the child.
type terminalExecResult struct {
	ExitCode int
	Error    string
}

// HandleTerminalExec must run first in ddev's main(). In a child started by
// ExecInChild, it runs the exec, writes its result, and exits with its status.
func HandleTerminalExec() {
	if len(os.Args) != 3 || os.Args[1] != terminalExecArg {
		canExecInChild = true
		return
	}
	dir := os.Args[2]
	exitCode, err := runTerminalExec(filepath.Join(dir, terminalExecOptionsFile))
	result := terminalExecResult{ExitCode: exitCode}
	if err != nil {
		result.Error = err.Error()
	}
	if data, marshalErr := json.Marshal(result); marshalErr == nil {
		_ = os.WriteFile(filepath.Join(dir, terminalExecResultFile), data, 0600)
	}
	switch {
	case exitCode != 0:
		os.Exit(exitCode)
	case err != nil:
		os.Exit(1)
	}
	os.Exit(0)
}

// ExecWithStreams runs a compose exec that reads ddev's stdin and writes to
// stdout and stderr. When stdin is a terminal, it runs the exec in a child
// process; see terminalExecArg.
func ExecWithStreams(projectName string, opts api.RunOptions, stdout, stderr io.Writer) error {
	if !canExecInChild || !isatty.IsTerminal(os.Stdin.Fd()) {
		restore, err := SetExecStdin(os.Stdin)
		if err != nil {
			return err
		}
		defer restore()
		ctx, svc, err := NewComposeServiceWithStreams(stdout, stderr)
		if err != nil {
			return err
		}
		return ExitCodeToError(svc.Exec(ctx, projectName, opts))
	}
	ddevPath, err := os.Executable()
	if err != nil {
		return err
	}
	return ExecInChild(ddevPath, projectName, opts, os.Stdin, stdout, stderr)
}

// ExecInChild runs a compose exec in `ddevPath __terminal-exec` and returns
// the error ExecWithStreams returns in-process. The options and the result
// pass through files, because the options can exceed the size limit of an
// argument or environment variable.
func ExecInChild(ddevPath, projectName string, opts api.RunOptions, stdin io.Reader, stdout, stderr io.Writer) error {
	dir, err := os.MkdirTemp("", "ddev-terminal-exec-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	payload, err := json.Marshal(terminalExec{ProjectName: projectName, Options: opts})
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dir, terminalExecOptionsFile), payload, 0600); err != nil {
		return err
	}
	child := exec.Command(ddevPath, terminalExecArg, dir)
	child.Stdin, child.Stdout, child.Stderr = stdin, stdout, stderr
	if err = child.Start(); err != nil {
		return err
	}
	release := killWithParent(child.Process)
	defer release()
	// The interrupt handler exits without running deferred calls, and a signal
	// sent only to ddev, such as `kill`, would leave the child running.
	unregister := interrupt.OnInterrupt(func(context.Context) {
		_ = child.Process.Kill()
		_ = os.RemoveAll(dir)
	})
	runErr := child.Wait()
	unregister()
	var result terminalExecResult
	if data, readErr := os.ReadFile(filepath.Join(dir, terminalExecResultFile)); readErr == nil && json.Unmarshal(data, &result) == nil {
		var resultErr error
		if result.Error != "" {
			resultErr = errors.New(result.Error)
		}
		return ExitCodeToError(result.ExitCode, resultErr)
	}
	// The child ended without a result, for example on a signal.
	if exitErr, ok := errors.AsType[*exec.ExitError](runErr); ok {
		return ExitCodeToError(exitErr.ExitCode(), nil)
	}
	return runErr
}

// runTerminalExec runs the exec in optionsPath. The child needs none of what
// getDockerManagerInstance adds to newDockerCli, such as plugins and server
// info.
func runTerminalExec(optionsPath string) (int, error) {
	data, err := os.ReadFile(optionsPath)
	if err != nil {
		return 0, err
	}
	// The options hold the exec's environment, so don't keep them on disk
	// while a long exec, such as `ddev ssh`, runs.
	_ = os.Remove(optionsPath)
	var te terminalExec
	if err = json.Unmarshal(data, &te); err != nil {
		return 0, err
	}
	dockerCli, _, err := newDockerCli(os.Stdin)
	if err != nil {
		return 0, err
	}
	svc, err := compose.NewComposeService(dockerCli, compose.WithOutputStream(os.Stdout), compose.WithErrorStream(os.Stderr))
	if err != nil {
		return 0, err
	}
	return svc.Exec(context.Background(), te.ProjectName, te.Options)
}
