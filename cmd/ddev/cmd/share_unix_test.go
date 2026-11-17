//go:build !windows

package cmd

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStopShareProvider(t *testing.T) {
	for _, tc := range []struct {
		name        string
		trap        string
		wantCleanup bool
	}{
		{name: "runs cleanup", trap: `trap 'touch "$1"; exit 0' TERM`, wantCleanup: true},
		{name: "kills an unresponsive provider", trap: `trap '' TERM`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "cleanup")
			cmd := exec.Command("sh", "-c", tc.trap+`; echo ready; while :; do sleep 1; done`, "provider", marker)
			setProcessGroupAttr(cmd)
			stdout, err := cmd.StdoutPipe()
			require.NoError(t, err)
			require.NoError(t, cmd.Start())
			t.Cleanup(func() { killProcessTree(cmd) })
			ready := make(chan string, 1)
			go func() {
				line, _ := bufio.NewReader(stdout).ReadString('\n')
				ready <- line
			}()
			select {
			case line := <-ready:
				require.Equal(t, "ready\n", line)
			case <-time.After(5 * time.Second):
				t.Fatal("provider did not become ready")
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			grace := 2 * time.Second
			if !tc.wantCleanup {
				grace = 100 * time.Millisecond
			}
			err = stopShareProvider(cmd, done, grace)
			if tc.wantCleanup {
				require.NoError(t, err)
				require.FileExists(t, marker)
			} else {
				require.Error(t, err)
				require.NoFileExists(t, marker)
			}
		})
	}
}
