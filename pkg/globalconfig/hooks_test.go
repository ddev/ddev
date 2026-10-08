package globalconfig_test

import (
	"testing"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"
)

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
