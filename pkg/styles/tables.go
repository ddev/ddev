package styles

import (
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/jedib0t/go-pretty/v6/table"
)

var DefaultTableStyle = "StyleLight"

// For descriptions of the various styles, see
// https://github.com/jedib0t/go-pretty/blob/main/table/style.go

// Get a style by name, returning the default if not found
func GetTableStyle(name string) (style table.Style) {
	if _, ok := globalconfig.StyleMap[name]; ok {
		style = globalconfig.StyleMap[name]
		return style
	}
	return globalconfig.StyleMap[DefaultTableStyle]
}

// SimpleFormattingRequired() returns true if we should not be colorizing/styling text
func SimpleFormattingRequired() bool {
	if globalconfig.DdevGlobalConfig.SimpleFormatting {
		return true
	}
	return false
}

// SetGlobalTableStyle sets the table style to the globally configured style
func SetGlobalTableStyle(writer table.Writer) {
	styleName := globalconfig.GetTableStyle()
	style := GetTableStyle(styleName)
	if SimpleFormattingRequired() {
		style = GetTableStyle("default")
		style.Options.SeparateRows = false
		style.Options.SeparateFooter = false
		style.Options.SeparateColumns = false
		style.Options.SeparateHeader = false
		style.Options.DrawBorder = false
	}
	writer.SetStyle(style)
}
