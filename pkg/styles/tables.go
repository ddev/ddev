package styles

import (
	"github.com/jedib0t/go-pretty/v6/table"
)

var DefaultTableStyle = "StyleLight"

// For descriptions of the various styles, see
// https://github.com/jedib0t/go-pretty/blob/main/table/style.go

var OptionsSeparateRows = table.Options{
	DrawBorder:      true,
	SeparateColumns: true,
	SeparateFooter:  true,
	SeparateHeader:  true,
	SeparateRows:    true,
}

var StyleDescribe = table.Style{
	Name:    "StyleDescribe",
	HTML:    table.DefaultHTMLOptions,
	Box:     table.StyleBoxLight,
	Color:   table.ColorOptionsDefault,
	Format:  table.FormatOptionsDefault,
	Options: OptionsSeparateRows,
	Title:   table.TitleOptionsDefault,
}

var styleMap map[string]table.Style = map[string]table.Style{
	"StyleDefault":                    table.StyleDefault,
	"StyleBold":                       table.StyleBold,
	"StyleLight":                      table.StyleLight,
	"StyleDouble":                     table.StyleDouble,
	"StyleRounded":                    table.StyleRounded,
	"StyleColoredBright":              table.StyleColoredBright,
	"StyleColoredDark":                table.StyleColoredDark,
	"StyleColoredBlackOnBlueWhite":    table.StyleColoredBlackOnBlueWhite,
	"StyleColoredBlackOnCyanWhite":    table.StyleColoredBlackOnCyanWhite,
	"StyleColoredBlackOnGreenWhite":   table.StyleColoredBlackOnGreenWhite,
	"StyleColoredBlackOnMagentaWhite": table.StyleColoredBlackOnMagentaWhite,
	"StyleColoredBlackOnYellowWhite":  table.StyleColoredBlackOnYellowWhite,
	"StyleColoredBlackOnRedWhite":     table.StyleColoredBlackOnRedWhite,
	"StyleColoredBlueWhiteOnBlack":    table.StyleColoredBlueWhiteOnBlack,
	"StyleColoredCyanWhiteOnBlack":    table.StyleColoredCyanWhiteOnBlack,
	"StyleColoredGreenWhiteOnBlack":   table.StyleColoredGreenWhiteOnBlack,
	"StyleColoredMagentaWhiteOnBlack": table.StyleColoredMagentaWhiteOnBlack,
	"StyleColoredRedWhiteOnBlack":     table.StyleColoredRedWhiteOnBlack,
	"StyleColoredYellowWhiteOnBlack":  table.StyleColoredYellowWhiteOnBlack,
	"StyleDescribe":                   StyleDescribe,
}

// ValidTableStyleList returns an array of valid styles
func ValidTableStyleList() []string {
	list := []string{}
	for k := range styleMap {
		list = append(list, k)
	}
	return list
}

// IsValidTableStyle checks to see if the table style is valid
func IsValidTableStyle(style string) bool {
	if _, ok := styleMap[style]; ok || style == "" {
		return true
	}

	return false
}

// Get a style by name, returning the default if not found
func GetTableStyle(name string) (style table.Style) {
	if _, ok := styleMap[name]; ok {
		style = styleMap[name]
		return style
	}
	return styleMap[DefaultTableStyle]
}
