package globalconfig

import (
	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	DdevDefaultStyle = table.Style{
		Name:    "StyleLight",
		Box:     table.StyleBoxLight,
		Color:   table.ColorOptionsDefault,
		Format:  table.FormatOptionsDefault,
		HTML:    table.DefaultHTMLOptions,
		Options: OptionsSeparateRows,
		Title:   table.TitleOptionsDefault,
	}
)

// StyleMap give the list of available styles
var StyleMap map[string]table.Style = map[string]table.Style{
	"default": DdevDefaultStyle,
	"bold":    table.StyleBold,
	"bright":  table.StyleColoredBright,
	//"StyleDouble":                     table.StyleDouble,
	//"StyleRounded":                    table.StyleRounded,
	//"StyleColoredDark":                table.StyleColoredDark,
	//"StyleColoredBlackOnBlueWhite":    table.StyleColoredBlackOnBlueWhite,
	//"StyleColoredBlackOnCyanWhite":    table.StyleColoredBlackOnCyanWhite,
	//"StyleColoredBlackOnGreenWhite":   table.StyleColoredBlackOnGreenWhite,
	//"StyleColoredBlackOnMagentaWhite": table.StyleColoredBlackOnMagentaWhite,
	//"StyleColoredBlackOnYellowWhite":  table.StyleColoredBlackOnYellowWhite,
	//"StyleColoredBlackOnRedWhite":     table.StyleColoredBlackOnRedWhite,
	//"StyleColoredBlueWhiteOnBlack":    table.StyleColoredBlueWhiteOnBlack,
	//"StyleColoredCyanWhiteOnBlack":    table.StyleColoredCyanWhiteOnBlack,
	//"StyleColoredGreenWhiteOnBlack":   table.StyleColoredGreenWhiteOnBlack,
	//"StyleColoredMagentaWhiteOnBlack": table.StyleColoredMagentaWhiteOnBlack,
	//"StyleColoredRedWhiteOnBlack":     table.StyleColoredRedWhiteOnBlack,
	//"StyleColoredYellowWhiteOnBlack":  table.StyleColoredYellowWhiteOnBlack,
}

var OptionsSeparateRows = table.Options{
	DrawBorder:      true,
	SeparateColumns: true,
	SeparateFooter:  true,
	SeparateHeader:  true,
	SeparateRows:    true,
}

// IsValidTableStyle checks to see if the table style is valid
func IsValidTableStyle(style string) bool {
	if _, ok := StyleMap[style]; ok || style == "" {
		return true
	}

	return false
}

// ValidTableStyleList returns an array of valid styles
func ValidTableStyleList() []string {
	list := []string{}
	for k := range StyleMap {
		list = append(list, k)
	}
	return list
}
