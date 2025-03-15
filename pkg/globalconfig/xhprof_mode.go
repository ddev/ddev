package globalconfig

import (
	"github.com/ddev/ddev/pkg/config/types"
)

// GetXHProfMode returns the xhprof mode config respecting
// defaults.
func (c *GlobalConfig) GetXHProfMode() types.XHProfMode {
	switch {
	case c.XHProfMode == types.XHProfModeEmpty:
		return types.FlagXHProfModeDefault
	default:
		return c.XHProfMode
	}
}

// SetXHProfMode sets the xhprof mode config.
func (c *GlobalConfig) SetXHProfMode(xhprofMode string) *GlobalConfig {
	if types.IsValidPerformanceMode(xhprofMode, types.ConfigTypeGlobal) {
		c.XHProfMode = xhprofMode
	}
	return c
}
