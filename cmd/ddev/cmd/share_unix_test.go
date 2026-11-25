//go:build !windows

package cmd

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStopShareProvider(t *testing.T) {
	for _, tc := range []struct {
		name            string
		trap            string
		wantCleanup     bool
		wantExit        int
		secondInterrupt bool
	}{
		{name: "runs cleanup", trap: `trap 'touch "$1"; exit 0' TERM`, wantCleanup: true},
		{name: "cleanup exits with signal status", trap: `trap 'touch "$1"; exit 143' TERM`, wantCleanup: true, wantExit: 143},
		{name: "kills an unresponsive provider", trap: `trap '' TERM`, wantExit: -1},
		{name: "second interrupt kills immediately", trap: `trap '' TERM`, wantExit: -1, secondInterrupt: true},
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
			signals := make(chan os.Signal, 1)
			if tc.secondInterrupt {
				grace = 5 * time.Second
				signals <- os.Interrupt
			}
			started := time.Now()
			err = stopShareProvider(cmd, done, grace, signals)
			if tc.wantExit == 0 {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			if tc.secondInterrupt {
				require.Less(t, time.Since(started), 2*time.Second)
			}
			require.Equal(t, tc.wantExit, cmd.ProcessState.ExitCode())
			if tc.wantCleanup {
				require.FileExists(t, marker)
			} else {
				require.NoFileExists(t, marker)
			}
		})
	}
}
