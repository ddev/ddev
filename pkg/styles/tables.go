package styles

import (
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/term"
)

var DefaultTableStyle = "StyleLight"

// For descriptions of the various styles, see
// https://github.com/jedib0t/go-pretty/blob/main/table/style.go

// Get a style by name, returning the default if not found
func GetTableStyle(name string) (style table.Style) {
	if _, ok := globalconfig.StyleMap[name]; ok {
		style = globalconfig.StyleMap[name]
		if RequireSimpleFormatting() {
			style.Options.SeparateRows = false
			style.Options.SeparateFooter = false
			style.Options.SeparateColumns = false
			style.Options.SeparateHeader = false
			style.Options.DrawBorder = false
		}
		return style
	}
	return globalconfig.StyleMap[DefaultTableStyle]
}

// RequireSimpleFormatting() returns true if we should not be colorizing/styling text
func RequireSimpleFormatting() bool {
	if globalconfig.DdevGlobalConfig.SimpleFormatting || !term.IsTerminal(1) || !term.IsTerminal(0) {
		return true
	}
	return false
}

// SetGlobalTableStyle sets the table style to the globally configured style
func SetGlobalTableStyle(writer table.Writer) {
	styleName := globalconfig.GetTableStyle()
	style := GetTableStyle(styleName)
	writer.SetStyle(style)
}
