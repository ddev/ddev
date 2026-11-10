package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindDdev(t *testing.T) {
	t.Setenv("DDEV_BINARY", "/opt/ddev/bin/ddev")
	b, err := findDdev()
	require.NoError(t, err)
	require.Equal(t, "/opt/ddev/bin/ddev", b)

	dir := t.TempDir()
	fake := filepath.Join(dir, "ddev")
	require.NoError(t, os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("DDEV_BINARY", "")
	t.Setenv("PATH", dir)
	b, err = findDdev()
	require.NoError(t, err)
	require.Equal(t, fake, b)

	t.Setenv("PATH", t.TempDir())
	_, err = findDdev()
	require.ErrorContains(t, err, "ddev was not found on PATH")
}

func TestUseLoginShellPath(t *testing.T) {
	// Startup files often print, such as "Agent pid 1234" from ssh-agent.
	shell := filepath.Join(t.TempDir(), "noisy-shell")
	body := "#!/bin/sh\necho 'Agent pid 1234'\nexport PATH=/opt/login/bin:/usr/bin:/bin\nshift\nexec /bin/sh \"$@\"\n"
	require.NoError(t, os.WriteFile(shell, []byte(body), 0o755))
	t.Setenv("SHELL", shell)
	t.Setenv("PATH", "/usr/bin:/bin")
	useLoginShellPath()
	require.Equal(t, "/opt/login/bin:/usr/bin:/bin", os.Getenv("PATH"))

	t.Setenv("SHELL", filepath.Join(t.TempDir(), "missing"))
	useLoginShellPath()
	require.Equal(t, "/opt/login/bin:/usr/bin:/bin", os.Getenv("PATH"))
}

func TestVersionMismatches(t *testing.T) {
	dir := t.TempDir()
	script := func(name, body string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755))
		return p
	}
	same := script("same", "echo 'ddev version v1.25.5'")
	older := script("older", "echo 'A notice'; echo 'ddev version v1.24.10'")
	silent := script("silent", "true")

	require.Empty(t, versionMismatches("v1.25.5", []string{same, "", same}))
	bad := versionMismatches("v1.25.5", []string{same, older, silent, filepath.Join(dir, "missing")})
	require.Len(t, bad, 3)
	require.Equal(t, older+" is v1.24.10", bad[0])
	require.Equal(t, silent+" printed no version", bad[1])
	require.Contains(t, bad[2], "missing")
}

func TestStatusMark(t *testing.T) {
	require.Equal(t, "●", statusMark("running"))
	require.Equal(t, "○", statusMark("stopped"))
	require.Equal(t, "○", statusMark("paused"))
	require.Equal(t, "◌", statusMark("unhealthy"))
}
