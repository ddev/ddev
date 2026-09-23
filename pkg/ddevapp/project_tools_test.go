package ddevapp

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectToolsDockerfile(t *testing.T) {
	app, err := NewApp(t.TempDir(), false)
	require.NoError(t, err)
	app.Name = "project-tools-test"
	customDir := app.GetConfigPath("web-build")
	require.NoError(t, os.MkdirAll(customDir, 0755))
	custom := "RUN echo user-customization\n"
	require.NoError(t, os.WriteFile(filepath.Join(customDir, "Dockerfile"), []byte(custom), 0644))

	tools := []string{
		"/usr/local/bin/wp-cli", "/usr/local/bin/magerun\n", "/usr/local/bin/magerun2\n",
		"/usr/local/bin/drush8", "/var/tmp/backdrop_drush_commands", "symfony-cli", "/usr/local/bin/shopware-cli",
	}
	cases := []struct {
		projectType string
		want        []int
	}{
		{"wordpress", []int{0}},
		{"wp-bedrock", []int{0}},
		{"magento", []int{1}},
		{"magento2", []int{2}},
		{"drupal6", []int{3}},
		{"drupal7", []int{3}},
		{"backdrop", []int{3, 4}},
		{"symfony", []int{5}},
		{"shopware6", []int{6}},
		{"drupal", nil},
		{"drupal8", nil},
		{"drupal9", nil},
		{"drupal10", nil},
		{"drupal11", nil},
		{"drupal12", nil},
		{"php", nil},
	}
	for _, tc := range cases {
		t.Run(tc.projectType, func(t *testing.T) {
			app.Type = tc.projectType
			_, err := app.RenderComposeYAML()
			require.NoError(t, err)
			data, err := os.ReadFile(app.GetConfigPath(".webimageBuild/Dockerfile"))
			require.NoError(t, err)
			content := string(data)
			for i, tool := range tools {
				if slices.Contains(tc.want, i) {
					require.Contains(t, content, tool)
					require.Less(t, strings.Index(content, tool), strings.Index(content, custom))
				} else {
					require.NotContains(t, content, tool)
				}
			}
			require.Contains(t, content, custom)
		})
	}
}
