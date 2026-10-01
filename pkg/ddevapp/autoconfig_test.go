package ddevapp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestLooksLikeWebProject checks which directories `ddev start` offers to configure.
func TestLooksLikeWebProject(t *testing.T) {
	cases := map[string]struct {
		files []string
		want  bool
	}{
		"empty":           {nil, false},
		"notes only":      {[]string{"notes.txt", "src/main.go"}, false},
		"composer.json":   {[]string{"composer.json"}, true},
		"package.json":    {[]string{"package.json"}, true},
		"root index.html": {[]string{"index.html"}, true},
		"docroot index":   {[]string{"public/index.php"}, true},
		"drupal7":         {[]string{"misc/ajax.js"}, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dir := testcommon.CreateTmpDir("TestLooksLikeWebProject")
			t.Cleanup(func() { testcommon.CleanupDir(dir) })
			for _, f := range tc.files {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(f)), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0644))
			}
			app, err := ddevapp.NewAutoConfigApp(dir)
			require.NoError(t, err)
			require.Equal(t, tc.want, app.LooksLikeWebProject())
			require.NoDirExists(t, filepath.Join(dir, ".ddev"), "previewing a configuration must not write one")
		})
	}
}

// TestAutoConfigProblem checks that a project name that can't be used is reported.
func TestAutoConfigProblem(t *testing.T) {
	dir := filepath.Join(testcommon.CreateTmpDir("TestAutoConfigProblem"), "bad.name!")
	t.Cleanup(func() { testcommon.CleanupDir(filepath.Dir(dir)) })
	require.NoError(t, os.MkdirAll(dir, 0755))

	app, err := ddevapp.NewAutoConfigApp(dir)
	require.NoError(t, err)
	require.Contains(t, app.AutoConfigProblem(), "is not a valid project name")
}
