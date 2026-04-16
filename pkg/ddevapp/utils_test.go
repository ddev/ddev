package ddevapp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInactiveProjects(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	app := &ddevapp.DdevApp{}

	site := TestSites[0]

	t.Cleanup(func() {
		err := os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
	})
	// Make sure we have one app in existence
	err := app.Init(site.Dir)
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)

	// Stop all sites
	_, err = exec.RunCommand(DdevBin, []string{"stop", "--all"})
	require.NoError(t, err)

	apps, err := ddevapp.GetInactiveProjects()
	require.NoError(t, err)

	assert.Greater(len(apps), 0)
}

func TestExtractProjectNames(t *testing.T) {
	var apps []*ddevapp.DdevApp

	assert := asrt.New(t)

	apps = append(apps, &ddevapp.DdevApp{Name: "bar"})
	apps = append(apps, &ddevapp.DdevApp{Name: "foo"})
	apps = append(apps, &ddevapp.DdevApp{Name: "zoz"})

	names := ddevapp.ExtractProjectNames(apps)
	expectedNames := []string{"bar", "foo", "zoz"}

	assert.Equal(expectedNames, names)
}

// TestCheckForConf validates CheckForConf behavior including nested project detection
func TestCheckForConf(t *testing.T) {
	// Use a subdirectory structure: parentDir/childDir
	parentDir := t.TempDir()
	childDir := filepath.Join(parentDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))

	parentConf := filepath.Join(parentDir, ".ddev", "config.yaml")
	childConf := filepath.Join(childDir, ".ddev", "config.yaml")

	// No config anywhere: expect error
	_, err := ddevapp.CheckForConf(childDir)
	require.Error(t, err)

	// Config only in parent: expect parent returned, no warning
	require.NoError(t, os.MkdirAll(filepath.Dir(parentConf), 0755))
	require.NoError(t, os.WriteFile(parentConf, []byte(""), 0644))

	got, err := ddevapp.CheckForConf(childDir)
	require.NoError(t, err)
	require.Equal(t, parentDir, got)

	// Config in both parent and child (nested project): expect child returned with warning
	require.NoError(t, os.MkdirAll(filepath.Dir(childConf), 0755))
	require.NoError(t, os.WriteFile(childConf, []byte(""), 0644))

	restoreOutput := util.CaptureUserErr()
	got, err = ddevapp.CheckForConf(childDir)
	out := restoreOutput()

	require.NoError(t, err)
	require.Equal(t, childDir, got)
	require.True(t, strings.Contains(out, "Nested project"), "expected nested project warning, got: %q", out)
}

// TestGetRelativeWorkingDirectory validates GetRelativeWorkingDirectory
func TestGetRelativeWorkingDirectory(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	app := &ddevapp.DdevApp{}

	site := TestSites[0]

	t.Cleanup(func() {
		err := os.Chdir(origDir)
		assert.NoError(err)
		err = app.Stop(true, false)
		assert.NoError(err)
		_ = os.RemoveAll("one")
	})
	// Make sure we have one app in existence
	err := app.Init(site.Dir)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(app.AppRoot, "one", "two", "three", "four"), 0755)
	require.NoError(t, err)

	testCases := []string{filepath.Join("one"), filepath.Join("one", "two"), filepath.Join("one", "two", "three"), filepath.Join("one", "two", "three", "four")}
	for _, c := range testCases {
		err = os.Chdir(filepath.Join(app.AppRoot, c))
		require.NoError(t, err)
		relDir := app.GetRelativeWorkingDirectory()
		assert.Equal(filepath.ToSlash(c), filepath.ToSlash(relDir))
	}
}
