package ddevapp

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// TestWebEnvironmentRendering tests that web_environment entries reach the
// rendered compose YAML unchanged, whatever characters they contain
func TestWebEnvironmentRendering(t *testing.T) {
	t.Setenv("DDEV_XDG_CONFIG_HOME", t.TempDir())
	app, err := NewApp(t.TempDir(), false)
	require.NoError(t, err)
	app.Name = "web-environment-test"

	entries := []string{
		`QUOTED="clientid"`,
		`WINDOWS_PATH=C:\Users\new\table`,
		"MULTILINE=line1\nline2",
		"PASSWORD=p3s6v9y$$B?E(H+MbQeT)",
		"INTERPOLATED=${DDEV_SITENAME}",
		"SYMBOLS=<a & b> #not-a-comment: {x}",
		"BARE_NAME",
	}
	app.WebEnvironment = entries

	rendered, err := app.RenderComposeYAML()
	require.NoError(t, err)

	var compose struct {
		Services map[string]struct {
			Environment []string `yaml:"environment"`
		} `yaml:"services"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(rendered), &compose), rendered)
	for _, entry := range entries {
		require.Contains(t, compose.Services["web"].Environment, entry)
	}
}
