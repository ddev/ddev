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

	known := t.TempDir()
	orig := wellKnownDirs
	t.Cleanup(func() { wellKnownDirs = orig })
	wellKnownDirs = func() []string { return []string{known} }
	t.Setenv("PATH", t.TempDir())
	_, err = findDdev()
	require.ErrorContains(t, err, "ddev was not found next to this app or on PATH")

	// A login shell without Homebrew on its PATH still finds Homebrew's ddev.
	brewed := filepath.Join(known, "ddev")
	require.NoError(t, os.WriteFile(brewed, []byte("#!/bin/sh\n"), 0o755))
	b, err = findDdev()
	require.NoError(t, err)
	require.Equal(t, brewed, b)
}

func TestDdevBeside(t *testing.T) {
	dir := t.TempDir()
	macos := filepath.Join(dir, "DDEV.app", "Contents", "MacOS")
	require.NoError(t, os.MkdirAll(macos, 0o755))
	require.Empty(t, ddevBeside(macos))

	// As `ddev utility download-ddev` leaves them: ddev next to DDEV.app.
	beside := filepath.Join(dir, "ddev")
	require.NoError(t, os.WriteFile(beside, []byte("#!/bin/sh\n"), 0o755))
	require.Equal(t, beside, ddevBeside(macos))
	require.Equal(t, beside, ddevBeside(dir), "a Linux or Windows app next to ddev")

	// A ddev inside the bundle wins.
	inside := filepath.Join(macos, "ddev")
	require.NoError(t, os.WriteFile(inside, []byte("#!/bin/sh\n"), 0o755))
	require.Equal(t, inside, ddevBeside(macos))
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

func TestStatusMark(t *testing.T) {
	require.Equal(t, "●", statusMark("running"))
	require.Equal(t, "○", statusMark("stopped"))
	require.Equal(t, "○", statusMark("paused"))
	require.Equal(t, "◌", statusMark("unhealthy"))
}
