package ddevapp_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// fakeDdevHostname stands in for ddev-hostname, appending each name to $FAKE_HOSTS_FILE.
// The sleep between read and write makes unserialized runs lose each other's names.
const fakeDdevHostname = `#!/bin/sh
[ "$1" = --version ] && exit 0
content=$(cat "$FAKE_HOSTS_FILE")
sleep 0.05
printf '%s\n%s\n' "$content" "$1" >"$FAKE_HOSTS_FILE"
`

// TestConcurrentHostsEntries checks that concurrent AddHostsEntriesIfNeeded
// calls from several projects don't lose each other's hosts-file entries.
func TestConcurrentHostsEntries(t *testing.T) {
	if nodeps.IsWindows() || nodeps.IsWSL2() {
		t.Skip("The fake ddev-hostname is a shell script, and Windows and WSL2 use ddev-hostname.exe")
	}
	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "ddev-hostname"), []byte(fakeDdevHostname), 0755))
	hostsFile := filepath.Join(t.TempDir(), "hosts")
	require.NoError(t, os.WriteFile(hostsFile, nil, 0644))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_HOSTS_FILE", hostsFile)
	t.Setenv("DDEV_NONINTERACTIVE", "")

	var apps []*ddevapp.DdevApp
	for i := range 6 {
		app := &ddevapp.DdevApp{
			Name:               fmt.Sprintf("hostsrace%d", i),
			ProjectTLD:         nodeps.DdevDefaultTLD,
			UseDNSWhenPossible: false,
		}
		for j := range 4 {
			app.AdditionalHostnames = append(app.AdditionalHostnames, fmt.Sprintf("h%d-hostsrace%d", j, i))
		}
		apps = append(apps, app)
	}

	var wg sync.WaitGroup
	errs := make([]error, len(apps))
	for i, app := range apps {
		wg.Go(func() { errs[i] = app.AddHostsEntriesIfNeeded() })
	}
	wg.Wait()

	b, err := os.ReadFile(hostsFile)
	require.NoError(t, err)
	written := strings.Fields(string(b))
	for i, app := range apps {
		require.NoError(t, errs[i], "project %s", app.Name)
		for _, name := range app.GetHostnames() {
			require.Contains(t, written, name)
		}
	}
}
