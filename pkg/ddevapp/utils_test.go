package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
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

// TestCheckForConf validates that CheckForConf finds .ddev/config.yaml in the given
// directory or in the closest parent that has one.
func TestCheckForConf(t *testing.T) {
	projectDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() { testcommon.CleanupDir(projectDir) })
	subDir := filepath.Join(projectDir, "web", "sites")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	_, err := ddevapp.CheckForConf(subDir)
	require.Error(t, err)

	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, ".ddev"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, ".ddev", "config.yaml"), nil, 0644))

	for _, dir := range []string{projectDir, subDir} {
		got, err := ddevapp.CheckForConf(dir)
		require.NoError(t, err)
		require.Equal(t, projectDir, got)
	}
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
