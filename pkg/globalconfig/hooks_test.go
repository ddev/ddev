package globalconfig_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"
)

func TestSchemasListEveryValidHookName(t *testing.T) {
	for _, schemaPath := range []string{"schema.json", filepath.Join("..", "ddevapp", "schema.json")} {
		raw, err := os.ReadFile(schemaPath)
		require.NoError(t, err)
		var schema struct {
			Properties struct {
				Hooks struct {
					Properties map[string]any `json:"properties"`
				} `json:"hooks"`
			} `json:"properties"`
		}
		require.NoError(t, json.Unmarshal(raw, &schema))
		for _, hookName := range globalconfig.ValidHookNames {
			require.Contains(t, schema.Properties.Hooks.Properties, hookName, "%s is missing hook %s", schemaPath, hookName)
		}
	}
}

func TestValidateHooks(t *testing.T) {
	require.NoError(t, globalconfig.ValidateHooks(map[string][]map[string]any{
		"post-start": {{"exec": "x"}, {"exec-host": "y"}, {"composer": "z"}},
	}))
	require.ErrorContains(t, globalconfig.ValidateHooks(map[string][]map[string]any{"post-strat": {{"exec": "x"}}}), "invalid hook post-strat")
	require.ErrorContains(t, globalconfig.ValidateHooks(map[string][]map[string]any{"post-start": {{"bogus": "x"}}}), "invalid task")
}

func TestValidateGlobalConfigHooks(t *testing.T) {
	orig := globalconfig.DdevGlobalConfig.Hooks
	t.Cleanup(func() { globalconfig.DdevGlobalConfig.Hooks = orig })
	globalconfig.DdevGlobalConfig.Hooks = map[string][]map[string]any{"post-strat": {{"exec": "x"}}}
	err := globalconfig.ValidateGlobalConfig()
	require.ErrorContains(t, err, "invalid hook post-strat")
	require.ErrorContains(t, err, globalconfig.GetGlobalConfigPath())
}
