// Package ansistyles is a complete port of the "ansi-styles" node.js library.
//
// This library just generates ANSI escape codes - it does nothing to decide whether
// the current terminal is capable of showing the generated codes.  You probably
// want a higher-level module for styling your strings.
//
// Basic usage:
//
//     fmt.Println(ansistyles.Green.Open + "Hello world!" + ansistyles.Green.Close)
//
// Each "style" has an "Open" and "Close" property.
//
// Modifiers:
//
// - `Reset`
// - `Bold`
// - `Dim`
// - `Italic` _(Not widely supported)_
// - `Underline`
// - `Overline` _Supported on VTE-based terminals, the GNOME terminal, mintty, and Git Bash._
// - `Inverse`
// - `Hidden`
// - `Strikethrough` _(Not widely supported)_
//
// Colors:
//
// - `Black`
// - `Red`
// - `Green`
// - `Yellow`
// - `Blue`
// - `Magenta`
// - `Cyan`
// - `White`
// - `BrightBlack` (alias: `Gray`, `Grey`)
// - `BrightRed`
// - `BrightGreen`
// - `BrightYellow`
// - `BrightBlue`
// - `BrightMagenta`
// - `BrightCyan`
// - `BrightWhite`
//
// Background colors:
//
// - `BgBlack`
// - `BgRed`
// - `BgGreen`
// - `BgYellow`
// - `BgBlue`
// - `BgMagenta`
// - `BgCyan`
// - `BgWhite`
// - `BgBrightBlack` (alias: `BgGray`, `BgGrey`)
// - `BgBrightRed`
// - `BgBrightGreen`
// - `BgBrightYellow`
// - `BgBrightBlue`
// - `BgBrightMagenta`
// - `BgBrightCyan`
// - `BgBrightWhite`
//
// Styles are available directly as values (e.g. `ansistyles.Blue`), via a
// lookup map using string names (e.g. `ansistyles.Styles["blue"]`), and also by
// grouped maps (e.g. `ansistyles.BgColor["bgBlue"]`):
//
// - `ansistyles.Modifier`
// - `ansistyles.Color`
// - `ansistyles.BgColor`
//
// Example:
//
//     fmt.Println(ansistyles.Color["green"].open);
//
// Raw escape codes (i.e. without the CSI escape prefix `\u001B[` and render mode
// postfix `m`) are available under `style.Codes`, which returns a `map` with the
// open codes as keys and close codes as values:
//
//     fmt.Println(ansistyles.Codes[36]); //=> 39
//
// `ansistyles` allows converting between various color formats and ANSI escapes,
// with support for 256 and 16 million colors.   The following color spaces are supported:
//
// - `rgb`
// - `hex`
// - `ansi256`
//
// To use these, call the associated conversion function with the intended output, for example:
//
//     ansistyles.Ansi256(ansistyles.RGBToAnsi256(100, 200, 15)) // RGB to 256 color ansi foreground code
//     ansistyles.BgAnsi256(ansistyles.HexToAnsi256("#C0FFEE")) // HEX to 256 color ansi foreground code
//
//     ansistyles.Ansi16m(100, 200, 15); // RGB to 16 million color foreground code
//     ansistyles.BgAnsi16m(ansistyles.HexToRgb("#C0FFEE")); // Hex (RGB) to 16 million color foreground code
//
// The node.js "ansi-styles" was originally written by Sindre Sorhus and Josh Junon,
// and is part of the popular node.js "chalk" library.
//
package ansistyles

import (
	"fmt"
	"math"
	"regexp"
)

// CSPair contains the ANSI color codes to open and close a given color.
type CSPair struct {
	Open  string
	Close string
}

func namedCSPair(open uint8, close uint8) CSPair {
	codes[open] = close
	return CSPair{
		Open:  fmt.Sprintf("\u001B[%dm", open),
		Close: fmt.Sprintf("\u001B[%dm", close),
	}
}

// Ansi returns the string used to set the foreground color, based on a basic 16-color Ansi code.
//
// `color` should be a number between 30 and 37 or 90 and 97, inclusive.
func Ansi(color uint8) string {
	return fmt.Sprintf("\u001B[%dm", color)
}

// BgAnsi returns the string used to set the background color, based on a basic 16-color Ansi code.
//
// `color` should be a number between 30 and 37 or 90 and 97, inclusive.
func BgAnsi(color uint8) string {
	return fmt.Sprintf("\u001B[%dm", color+10)
}

// Ansi256 returns the string used to set the foreground color, based on Ansi 256 color lookup table.
//
// See https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit.
func Ansi256(color uint8) string {
	return fmt.Sprintf("\u001B[38;5;%dm", color)
}

// BgAnsi256 returns the string used to set the background color, based on Ansi 256 color lookup table.
//
// See https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit.
func BgAnsi256(color uint8) string {
	return fmt.Sprintf("\u001B[48;5;%dm", color)
}

// Ansi16m returns the string used to set a 24bit foreground color.
func Ansi16m(red uint8, green uint8, blue uint8) string {
	return fmt.Sprintf("\u001B[38;2;%v;%v;%vm", red, green, blue)
}

// BgAnsi16m returns the string used to set a 24bit background color.
func BgAnsi16m(red uint8, green uint8, blue uint8) string {
	return fmt.Sprintf("\u001B[48;2;%v;%v;%vm", red, green, blue)
}

// Close is the "close" code for 256 amd 16m ansi color codes.
const Close = "\u001B[39m"

// BgClose is the "close" code for 256 amd 16m ansi background color codes.
const BgClose = "\u001B[49m"

// Reset colors
var Reset = namedCSPair(0, 0)

// Bold modifier
// 21 isn't widely supported and 22 does the same thing
var Bold = namedCSPair(1, 22)

// Dim modifier
var Dim = namedCSPair(2, 22)

// Italic modifier
var Italic = namedCSPair(3, 23)

// Underline modifier
var Underline = namedCSPair(4, 24)

// Overline modifier
var Overline = namedCSPair(53, 55)

// Inverse modifier
var Inverse = namedCSPair(7, 27)

// Hidden modifier
var Hidden = namedCSPair(8, 28)

// Strikethrough modifier
var Strikethrough = namedCSPair(9, 29)

// Black foreground color
var Black = namedCSPair(30, 39)

// Red foreground color
var Red = namedCSPair(31, 39)

// Green foreground color
var Green = namedCSPair(32, 39)

// Yellow foreground color
var Yellow = namedCSPair(33, 39)

// Blue foreground color
var Blue = namedCSPair(34, 39)

// Magenta foreground color
var Magenta = namedCSPair(35, 39)

// Cyan foreground color
var Cyan = namedCSPair(36, 39)

// White foreground color
var White = namedCSPair(37, 39)

// BrightBlack foreground color
var BrightBlack = namedCSPair(90, 39)

// BrightRed foreground color
var BrightRed = namedCSPair(91, 39)

// BrightGreen foreground color
var BrightGreen = namedCSPair(92, 39)

// BrightYellow foreground color
var BrightYellow = namedCSPair(93, 39)

// BrightBlue foreground color
var BrightBlue = namedCSPair(94, 39)

// BrightMagenta foreground color
var BrightMagenta = namedCSPair(95, 39)

// BrightCyan foreground color
var BrightCyan = namedCSPair(96, 39)

// BrightWhite foreground color
var BrightWhite = namedCSPair(97, 39)

// Grey is an alias for BrightBlack
var Grey = BrightBlack

// Gray is an alias for BrightBlack
var Gray = BrightBlack

// BgBlack background color
var BgBlack = namedCSPair(40, 49)

// BgRed background color
var BgRed = namedCSPair(41, 49)

// BgGreen background color
var BgGreen = namedCSPair(42, 49)

// BgYellow background color
var BgYellow = namedCSPair(43, 49)

// BgBlue background color
var BgBlue = namedCSPair(44, 49)

// BgMagenta background color
var BgMagenta = namedCSPair(45, 49)

// BgCyan background color
var BgCyan = namedCSPair(46, 49)

// BgWhite background color
var BgWhite = namedCSPair(47, 49)

// BgBrightBlack background color
var BgBrightBlack = namedCSPair(100, 49)

// BgBrightRed background color
var BgBrightRed = namedCSPair(101, 49)

// BgBrightGreen background color
var BgBrightGreen = namedCSPair(102, 49)

// BgBrightYellow background color
var BgBrightYellow = namedCSPair(103, 49)

// BgBrightBlue background color
var BgBrightBlue = namedCSPair(104, 49)

// BgBrightMagenta background color
var BgBrightMagenta = namedCSPair(105, 49)

// BgBrightCyan background color
var BgBrightCyan = namedCSPair(106, 49)

// BgBrightWhite background color
var BgBrightWhite = namedCSPair(107, 49)

// BgGrey is an alias for BgBrightBlack
var BgGrey = BgBrightBlack

// BgGray is an alias for BgBrightBlack
var BgGray = BgBrightBlack

// Styles is a map of colors and modifiers by name
var Styles = map[string]CSPair{
	"reset":           Reset,
	"bold":            Bold,
	"dim":             Dim,
	"italic":          Italic,
	"underline":       Underline,
	"overline":        Overline,
	"inverse":         Inverse,
	"hidden":          Hidden,
	"strikethrough":   Strikethrough,
	"black":           Black,
	"red":             Red,
	"green":           Green,
	"yellow":          Yellow,
	"blue":            Blue,
	"magenta":         Magenta,
	"cyan":            Cyan,
	"white":           White,
	"brightBlack":     BrightBlack,
	"brightRed":       BrightRed,
	"brightGreen":     BrightGreen,
	"brightYellow":    BrightYellow,
	"brightBlue":      BrightBlue,
	"brightMagenta":   BrightMagenta,
	"brightCyan":      BrightCyan,
	"brightWhite":     BrightWhite,
	"grey":            BrightBlack,
	"gray":            BrightBlack,
	"bgBlack":         BgBlack,
	"bgRed":           BgRed,
	"bgGreen":         BgGreen,
	"bgYellow":        BgYellow,
	"bgBlue":          BgBlue,
	"bgMagenta":       BgMagenta,
	"bgCyan":          BgCyan,
	"bgWhite":         BgWhite,
	"bgBrightBlack":   BgBrightBlack,
	"bgBrightRed":     BgBrightRed,
	"bgBrightGreen":   BgBrightGreen,
	"bgBrightYellow":  BgBrightYellow,
	"bgBrightBlue":    BgBrightBlue,
	"bgBrightMagenta": BgBrightMagenta,
	"bgBrightCyan":    BgBrightCyan,
	"bgBrightWhite":   BgBrightWhite,
	"bgGrey":          BgBrightBlack,
	"bgGray":          BgBrightBlack,
}

// Modifier is a map of modifiers by name
var Modifier = map[string]CSPair{
	"reset":         Reset,
	"bold":          Bold,
	"dim":           Dim,
	"italic":        Italic,
	"underline":     Underline,
	"overline":      Overline,
	"inverse":       Inverse,
	"hidden":        Hidden,
	"strikethrough": Strikethrough,
}

// Color is a map of colors by name
var Color = map[string]CSPair{
	"black":         Black,
	"red":           Red,
	"green":         Green,
	"yellow":        Yellow,
	"blue":          Blue,
	"magenta":       Magenta,
	"cyan":          Cyan,
	"white":         White,
	"brightBlack":   BrightBlack,
	"brightRed":     BrightRed,
	"brightGreen":   BrightGreen,
	"brightYellow":  BrightYellow,
	"brightBlue":    BrightBlue,
	"brightMagenta": BrightMagenta,
	"brightCyan":    BrightCyan,
	"brightWhite":   BrightWhite,
	"grey":          BrightBlack,
	"gray":          BrightBlack,
}

// BgColor is a map of background colors by name
var BgColor = map[string]CSPair{
	"bgBlack":         BgBlack,
	"bgRed":           BgRed,
	"bgGreen":         BgGreen,
	"bgYellow":        BgYellow,
	"bgBlue":          BgBlue,
	"bgMagenta":       BgMagenta,
	"bgCyan":          BgCyan,
	"bgWhite":         BgWhite,
	"bgBrightBlack":   BgBrightBlack,
	"bgBrightRed":     BgBrightRed,
	"bgBrightGreen":   BgBrightGreen,
	"bgBrightYellow":  BgBrightYellow,
	"bgBrightBlue":    BgBrightBlue,
	"bgBrightMagenta": BgBrightMagenta,
	"bgBrightCyan":    BgBrightCyan,
	"bgBrightWhite":   BgBrightWhite,
	"bgGrey":          BgBrightBlack,
	"bgGray":          BgBrightBlack,
}

// RGBToAnsi256 converts from the RGB color space to the ANSI 256 color space.
func RGBToAnsi256(red uint8, green uint8, blue uint8) uint8 {
	// Originally from // From https://github.com/Qix-/color-convert/blob/3f0e0d4e92e235796ccb17f6e85c72094a651f49/conversions.js

	// We use the extended greyscale palette here, with the exception of
	// black and white. normal palette only has 4 greyscale shades.
	if red == green && green == blue {
		if red < 8 {
			return 16
		}

		if red > 248 {
			return 231
		}

		return uint8(math.Round(((float64(red)-8)/247)*24)) + 232
	}

	return 16 +
		uint8(
			(36*math.Round(float64(red)/255*5))+
				(6*math.Round(float64(green)/255*5))+
				math.Round(float64(blue)/255*5))
}

var hexColorRegex = regexp.MustCompile(`([a-fA-F\d]{6}|[a-fA-F\d]{3})`)

// HexToRGB converts from the RGB HEX color space to the RGB color space.
//
// "hex" should be a hexadecimal string containing RGB data (e.g. "#2340ff" or "00f").
// Note that the leading "#" is optional.  If the hex code passed in is invalid,
// this will return 0, 0, 0 - it's up to you to validate the input if you want to
// detect invalid values.
func HexToRGB(hex string) (red uint8, green uint8, blue uint8) {
	matches := hexColorRegex.FindStringSubmatch(hex)
	if matches == nil {
		return 0, 0, 0
	}

	s := matches[0]

	// Adapted from https://stackoverflow.com/questions/54197913/parse-hex-string-to-image-color
	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		default:
			panic(fmt.Sprintf("Unhandled char %v", b))
		}
	}

	switch len(s) {
	case 6:
		red = hexToByte(s[0])<<4 + hexToByte(s[1])
		green = hexToByte(s[2])<<4 + hexToByte(s[3])
		blue = hexToByte(s[4])<<4 + hexToByte(s[5])
	case 3:
		red = hexToByte(s[0]) * 17
		green = hexToByte(s[1]) * 17
		blue = hexToByte(s[2]) * 17
	default:
		return 0, 0, 0
	}
	return
}

// HexToAnsi256 converts from the RGB HEX color space to the ANSI 256 color space.
// hex should be a hexadecimal string containing RGB data.
func HexToAnsi256(hex string) uint8 {
	return RGBToAnsi256(HexToRGB(hex))
}

// RGBToAnsi converts from the RGB color space to the ANSI 16 color space.
func RGBToAnsi(r uint8, g uint8, b uint8) uint8 {
	return Ansi256ToAnsi(RGBToAnsi256(r, g, b))
}

// Ansi256ToAnsi converts from the ANSI 256 color space to the ANSI 16 color space.
func Ansi256ToAnsi(code uint8) uint8 {
	return ansi256ToAnsi16Lut[code]
}

// codes is a map of raw escape codes (i.e. without the CSI escape prefix `\u001B[`
// and render mode postfix `m`), with the open codes as keys and close codes as values.
//
//     fmt.Println(ansistyles.Codes[36]); //=> 39
//
var codes map[uint8]uint8 = make(map[uint8]uint8, 41)

// CloseCode returns the raw close escape code (i.e. without the CSI escape prefix `\u001B[`
// and render mode postfix `m`), given a raw open code.
//
//     fmt.Println(ansistyles.CloseCode(36)); //=> 39
//
func CloseCode(code uint8) uint8 {
	return codes[code]
}
