package types

import (
	"fmt"
	"strings"
)

type XHProfMode = string

const (
	XHProfModeEmpty   XHProfMode = ""
	XHProfModePrepend XHProfMode = "prepend"
	XHProfModeXHGui   XHProfMode = "xhgui"
	XHProfModeGlobal  XHProfMode = "global"
)

// ValidXHProfModeOptions returns a slice of valid xhprof mode
// options for the project config or global config if global is true.
func ValidXHProfModeOptions(configType ConfigType) []XHProfMode {
	switch configType {
	case ConfigTypeGlobal:
		return []XHProfMode{
			XHProfModePrepend,
			XHProfModeXHGui,
		}
	case ConfigTypeProject:
		return []XHProfMode{
			XHProfModeGlobal,
			XHProfModePrepend,
			XHProfModeXHGui,
		}
	default:
		panic(fmt.Errorf("invalid ConfigType: %v", configType))
	}
}

// IsValidXHProfMode checks to see if the given xhprof mode
// option is valid.
func IsValidXHProfMode(XHProfMode string, configType ConfigType) bool {
	if XHProfMode == XHProfModeEmpty {
		return true
	}

	for _, option := range ValidXHProfModeOptions(configType) {
		if XHProfMode == option {
			return true
		}
	}

	return false
}

// CheckValidXHProfMode checks to see if the given xhprof mode option
// is valid and returns an error in case the value is not valid.
func CheckValidXHProfMode(xhprofMode string, configType ConfigType) error {
	if !IsValidXHProfMode(xhprofMode, configType) {
		return fmt.Errorf(
			"\"%s\" is not a valid xhprof_mode option. Valid options include \"%s\"",
			xhprofMode,
			strings.Join(ValidXHProfModeOptions(configType), "\", \""),
		)
	}

	return nil
}

// Flag definitions
const FlagXHProfModeName = "xhprof-mode"

// TODO: Default should change to XHProfModeXHGUI in DDEV v1.25.0
const FlagXHProfModeDefault = XHProfModePrepend

func FlagXHProfModeDescription(configType ConfigType) string {
	return fmt.Sprintf(
		"XHProf mode, possible values are \"%s\"",
		strings.Join(ValidXHProfModeOptions(configType), "\", \""),
	)
}

const FlagXHProfModeResetName = "xhprof-mode-reset"

func FlagXHProfModeResetDescription(configType ConfigType) string {
	switch configType {
	case ConfigTypeGlobal:
		return "Reset XHProf mode to default"
	case ConfigTypeProject:
		return "Reset XHProf mode to global configuration"
	default:
		panic(fmt.Errorf("invalid ConfigType: %v", configType))
	}
}
