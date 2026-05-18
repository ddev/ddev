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

// TestCheckForConf validates CheckForConf path-finding behavior
func TestCheckForConf(t *testing.T) {
	parentDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() { _ = os.RemoveAll(parentDir) })
	childDir := filepath.Join(parentDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))

	// No config anywhere: expect error
	_, err := ddevapp.CheckForConf(childDir)
	require.Error(t, err)

	// Config only in parent: expect parent returned
	require.NoError(t, os.MkdirAll(filepath.Join(parentDir, ".ddev"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(parentDir, ".ddev", "config.yaml"), []byte(""), 0644))
	got, err := ddevapp.CheckForConf(childDir)
	require.NoError(t, err)
	require.Equal(t, parentDir, got)

	// Config in child too: child returned (first match, early return)
	require.NoError(t, os.MkdirAll(filepath.Join(childDir, ".ddev"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(childDir, ".ddev", "config.yaml"), []byte(""), 0644))
	got, err = ddevapp.CheckForConf(childDir)
	require.NoError(t, err)
	require.Equal(t, childDir, got)
}

// TestDetectNestedProject validates DetectNestedProject behavior
func TestDetectNestedProject(t *testing.T) {
	parentDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() {
		_ = os.RemoveAll(parentDir)
	})
	childDir := filepath.Join(parentDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))

	// No config anywhere: not nested
	_, _, nested := ddevapp.DetectNestedProject(childDir)
	require.False(t, nested)

	// Config only in parent: not nested (single match is not "nested")
	require.NoError(t, os.MkdirAll(filepath.Join(parentDir, ".ddev"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(parentDir, ".ddev", "config.yaml"), []byte(""), 0644))
	_, _, nested = ddevapp.DetectNestedProject(childDir)
	require.False(t, nested)

	// Config only in child: not nested
	require.NoError(t, os.MkdirAll(filepath.Join(childDir, ".ddev"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(childDir, ".ddev", "config.yaml"), []byte(""), 0644))
	// Remove parent config first
	require.NoError(t, os.Remove(filepath.Join(parentDir, ".ddev", "config.yaml")))
	_, _, nested = ddevapp.DetectNestedProject(childDir)
	require.False(t, nested)

	// Config in both child and parent: nested
	require.NoError(t, os.WriteFile(filepath.Join(parentDir, ".ddev", "config.yaml"), []byte(""), 0644))
	c, p, nested := ddevapp.DetectNestedProject(childDir)
	require.True(t, nested)
	require.Equal(t, childDir, c)
	require.Equal(t, parentDir, p)

	// Config in child + grandparent (skipping immediate parent): nested
	grandparentDir := testcommon.CreateTmpDir(t.Name() + "-grandparent")
	t.Cleanup(func() {
		_ = os.RemoveAll(grandparentDir)
	})
	deepChildDir := filepath.Join(grandparentDir, "level1", "child")
	require.NoError(t, os.MkdirAll(deepChildDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(grandparentDir, ".ddev"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(grandparentDir, ".ddev", "config.yaml"), []byte(""), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(deepChildDir, ".ddev"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(deepChildDir, ".ddev", "config.yaml"), []byte(""), 0644))
	c, p, nested = ddevapp.DetectNestedProject(deepChildDir)
	require.True(t, nested)
	require.Equal(t, deepChildDir, c)
	require.Equal(t, grandparentDir, p)
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
