package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"
)

// TestEnvFileTarget tests which service each .env file name applies to
func TestEnvFileTarget(t *testing.T) {
	testCases := []struct {
		baseName string
		target   string
		ok       bool
	}{
		{".env", "", true},
		{".env.local", "", true},
		{".env.web", "web", true},
		{".env.web.local", "web", true},
		{".env.web.myaddon", "web", true},
		{".env.web.myaddon.local", "web", true},
		{".env.redis-build", "redis-build", true},
		// `local` is reserved, so it is never read as a service or a label here
		{".env.local.web", "local", true},
		{".env.example", "", false},
		{".env.web.example", "", false},
		{".envhello", "", false},
		{".env.", "", false},
		{"env.web", "", false},
		{".gitignore", "", false},
	}
	for _, tc := range testCases {
		target, ok := envFileTarget(tc.baseName)
		require.Equal(t, tc.ok, ok, "unexpected ok for %s", tc.baseName)
		require.Equal(t, tc.target, target, "unexpected target for %s", tc.baseName)
	}
}

// TestOrderedEnvFilesInDir tests the order .env files are applied in
func TestOrderedEnvFilesInDir(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		".env.web.myaddon", ".env.local", ".env.web", ".env.busybox.local",
		".env", ".env.busybox", ".env.db.local", ".env.example", ".env.web.example", ".envhello",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("A=b\n"), 0644))
	}
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".env.somedir"), 0755))

	envFiles, err := orderedEnvFilesInDir(dir)
	require.NoError(t, err)
	expected := []string{
		".env", ".env.local",
		".env.busybox", ".env.busybox.local",
		// .env.db.local applies even though there is no .env.db to override
		".env.db.local",
		".env.web", ".env.web.myaddon",
	}
	require.Equal(t, expected, baseNames(envFiles))

	envFiles, err = orderedEnvFilesInDir(filepath.Join(dir, "no-such-dir"))
	require.NoError(t, err)
	require.Empty(t, envFiles)
}

// TestFilterEnvFilesForTarget tests which of the ordered .env files each service gets
func TestFilterEnvFilesForTarget(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".env", ".env.local", ".env.busybox", ".env.db.local", ".env.web", ".env.web.myaddon"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("A=b\n"), 0644))
	}
	envFiles, err := orderedEnvFilesInDir(dir)
	require.NoError(t, err)

	require.Equal(t, []string{".env", ".env.local", ".env.web", ".env.web.myaddon"}, baseNames(filterEnvFilesForTarget(envFiles, "web")))
	require.Equal(t, []string{".env", ".env.local", ".env.db.local"}, baseNames(filterEnvFilesForTarget(envFiles, "db")))
	require.Equal(t, []string{".env", ".env.local"}, baseNames(filterEnvFilesForTarget(envFiles, "no-such-service")))
}

// TestEnvFilesGlobalAndProject tests that global .env files are applied before project ones
func TestEnvFilesGlobalAndProject(t *testing.T) {
	t.Setenv("DDEV_XDG_CONFIG_HOME", t.TempDir())
	globalDir := globalconfig.GetGlobalDdevDir()
	writeEnv := func(dir, name, contents string) {
		require.NoError(t, os.MkdirAll(dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(contents), 0644))
	}
	writeEnv(globalDir, ".env", "SHARED=global\nGLOBALONLY=yes\n")
	writeEnv(globalDir, ".env.web", "SHARED=globalweb\n")

	app := &DdevApp{Name: t.Name(), AppRoot: t.TempDir()}
	writeEnv(app.AppConfDir(), ".env", "SHARED=project\n")
	writeEnv(app.AppConfDir(), ".env.web.local", "SHARED=projectweblocal\n")

	envFiles, err := app.EnvFiles()
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join(globalDir, ".env"),
		filepath.Join(globalDir, ".env.web"),
		filepath.Join(app.AppConfDir(), ".env"),
		filepath.Join(app.AppConfDir(), ".env.web.local"),
	}, envFiles)

	webEnv, err := app.ReadEnvFilesForTarget("web")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"SHARED": "projectweblocal", "GLOBALONLY": "yes"}, webEnv)

	dbEnv, err := app.ReadEnvFilesForTarget("db")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"SHARED": "project", "GLOBALONLY": "yes"}, dbEnv)
}

// TestGitIgnoreEnvFiles tests that .env.local and .env.*.local stay ignored once they exist
func TestGitIgnoreEnvFiles(t *testing.T) {
	app := &DdevApp{Name: t.Name(), AppRoot: t.TempDir()}
	require.NoError(t, PrepDdevDirectory(app))
	gitIgnore := app.GetConfigPath(".gitignore")
	require.FileExists(t, gitIgnore)

	// A user-owned file drops out of .gitignore, but .env.local must not
	require.NoError(t, os.WriteFile(app.GetConfigPath(".env.local"), []byte("A=b\n"), 0644))
	require.NoError(t, os.WriteFile(app.GetConfigPath(".env.web.local"), []byte("A=b\n"), 0644))
	require.NoError(t, os.MkdirAll(app.GetConfigPath("apache"), 0755))
	require.NoError(t, os.WriteFile(app.GetConfigPath("apache/apache-site.conf"), []byte("# mine\n"), 0644))
	require.NoError(t, PrepDdevDirectory(app))

	contents, err := os.ReadFile(gitIgnore)
	require.NoError(t, err)
	require.Contains(t, string(contents), "\n/.env.local\n")
	require.Contains(t, string(contents), "\n/.env.*.local\n")
	require.NotContains(t, string(contents), "apache/apache-site.conf")

	globalGitIgnore, err := bundledAssets.ReadFile("global_dotddev_assets/.gitignore")
	require.NoError(t, err)
	require.Contains(t, string(globalGitIgnore), "\n/.env.local\n")
	require.Contains(t, string(globalGitIgnore), "\n/.env.*.local\n")
}

func baseNames(paths []string) []string {
	names := make([]string, 0, len(paths))
	for _, p := range paths {
		names = append(names, filepath.Base(p))
	}
	return names
}
