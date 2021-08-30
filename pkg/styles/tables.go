package styles

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/term"
	"os"
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

var (
	ddevDefault = table.Style{
		Name:    "StyleLight",
		Box:     table.StyleBoxLight,
		Color:   table.ColorOptionsDefault,
		Format:  table.FormatOptionsDefault,
		HTML:    table.DefaultHTMLOptions,
		Options: OptionsSeparateRows,
		Title:   table.TitleOptionsDefault,
	}
)

var styleMap map[string]table.Style = map[string]table.Style{
	"default": ddevDefault,
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
	//"StyleDescribe":                   StyleDescribe,
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
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			style.Options.SeparateRows = false
			style.Options.SeparateFooter = false
			style.Options.SeparateColumns = false
			style.Options.SeparateHeader = false
			style.Options.DrawBorder = false
		}
		return style
	}
	return styleMap[DefaultTableStyle]
}
