package cmd

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestInterruptRemovesTemporaryContainers checks that SIGTERM during
// `ddev start` removes the utility containers it has in flight.
func TestInterruptRemovesTemporaryContainers(t *testing.T) {
	// Sending a console Ctrl-C to a child needs GenerateConsoleCtrlEvent on Windows.
	if nodeps.IsWindows() {
		t.Skip("Skipping on Windows, which cannot send SIGTERM to a child process")
	}
	site := TestSites[0]
	t.Cleanup(site.Chdir())
	t.Cleanup(func() {
		_ = exec.Command(DdevBin, "stop").Run()
	})

	out, err := exec.Command(DdevBin, "stop").CombinedOutput()
	require.NoError(t, err, "output=%s", out)

	var output bytes.Buffer
	cmd := exec.Command(DdevBin, "start", "-y")
	cmd.Stdout = &output
	cmd.Stderr = &output
	require.NoError(t, cmd.Start())
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	seen := ""
	deadline := time.After(3 * time.Minute)
	for seen == "" {
		select {
		case err := <-exited:
			t.Fatalf("ddev start exited (err=%v) before a temporary container was seen; output=%s", err, output.String())
		case <-deadline:
			_ = cmd.Process.Kill()
			t.Fatalf("no temporary container appeared; output=%s", output.String())
		case <-time.After(50 * time.Millisecond):
		}
		names, err := temporaryContainerNames()
		require.NoError(t, err)
		if len(names) > 0 {
			seen = names[0]
		}
	}

	require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))
	err = <-exited
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr), "expected an exit error, got %v; output=%s", err, output.String())
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	require.True(t, ok)
	require.True(t, status.Signaled(), "ddev should die from SIGTERM so a calling shell sees it; status=%v output=%s", status, output.String())
	require.Equal(t, syscall.SIGTERM, status.Signal())

	names, err := temporaryContainerNames()
	require.NoError(t, err)
	require.Empty(t, names, "temporary containers left after SIGTERM (saw %s); output=%s", seen, output.String())
}

// temporaryContainerNames lists DDEV containers that are not project or
// global service containers, which are all named ddev-*.
func temporaryContainerNames() ([]string, error) {
	containers, err := dockerutil.FindContainersByLabels(map[string]string{"com.ddev.platform": "ddev"})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, c := range containers {
		for _, n := range c.Names {
			if !strings.HasPrefix(strings.TrimPrefix(n, "/"), "ddev-") {
				names = append(names, n)
			}
		}
	}
	return names, nil
}
