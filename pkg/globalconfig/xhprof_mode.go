package globalconfig //nolint:revive

import (
	"github.com/ddev/ddev/pkg/config/types"
)

// GetXHProfMode returns the xhprof mode config respecting
// defaults.
func (c *GlobalConfig) GetXHProfMode() types.XHProfMode {
	switch c.XHProfMode {
	case types.XHProfModeEmpty:
		return types.FlagXHProfModeDefault
	default:
		return c.XHProfMode
	}
}

// SetXHProfMode sets the xhprof mode config.
func (c *GlobalConfig) SetXHProfMode(xhprofMode string) *GlobalConfig {
	if types.IsValidXHProfMode(xhprofMode, types.ConfigTypeGlobal) {
		c.XHProfMode = xhprofMode
	}
	return c
}
