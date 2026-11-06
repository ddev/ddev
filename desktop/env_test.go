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

func TestStatusMark(t *testing.T) {
	require.Equal(t, "●", statusMark("running"))
	require.Equal(t, "○", statusMark("stopped"))
	require.Equal(t, "○", statusMark("paused"))
	require.Equal(t, "◌", statusMark("unhealthy"))
}
