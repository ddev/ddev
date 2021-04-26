package gchalk

import (
	"github.com/jwalton/gchalk/pkg/ansistyles"
)

// Ansi returns a function which colors a string using the ANSI 16 color pallette.
func Ansi(code uint8) func(strs ...string) string {
	return rootBuilder.WithAnsi(code).applyStyle
}

// WithAnsi returns a Builder that generates strings with the specified color.
func WithAnsi(code uint8) *Builder {
	return rootBuilder.WithAnsi(code)
}

// Ansi returns a function which colors a string using the ANSI 16 color pallette.
func (builder *Builder) Ansi(code uint8) func(strs ...string) string {
	return builder.WithAnsi(code).applyStyle
}

// WithAnsi returns a Builder that generates strings with the specified color.
func (builder *Builder) WithAnsi(code uint8) *Builder {
	closeCode := ansistyles.CloseCode(code)
	if closeCode == 0 {
		closeCode = 39
	}
	return createBuilder(builder, ansistyles.Ansi(code), ansistyles.Ansi(closeCode))
}

// BgAnsi returns a function which colors the background of a string using the ANSI 16 color pallette.
func BgAnsi(code uint8) func(strs ...string) string {
	return rootBuilder.WithBgAnsi(code).applyStyle
}

// WithBgAnsi returns a Builder that generates strings with the specified background color.
func WithBgAnsi(code uint8) *Builder {
	return rootBuilder.WithBgAnsi(code)
}

// BgAnsi returns a function which colors the background of a string using the ANSI 16 color pallette.
func (builder *Builder) BgAnsi(code uint8) func(strs ...string) string {
	return builder.WithBgAnsi(code).applyStyle
}

// WithBgAnsi returns a Builder that generates strings with the specified background color.
func (builder *Builder) WithBgAnsi(code uint8) *Builder {
	return createBuilder(builder, ansistyles.BgAnsi(code), ansistyles.BgClose)
}

// Ansi256 returns a function which colors a string using the ANSI 256 color pallette.
// If ANSI 256 color support is unavailable, this will automatically convert the color
// to the closest available color.
func Ansi256(code uint8) func(strs ...string) string {
	return rootBuilder.WithAnsi256(code).applyStyle
}

// WithAnsi256 returns a Builder that generates strings with the specified color.
// Note that if ANSI 256 support is unavailable, this will automatically
// convert the color to the closest available color.
func WithAnsi256(code uint8) *Builder {
	return rootBuilder.WithAnsi256(code)
}

// Ansi256 returns a function which colors a string using the ANSI 256 color pallette.
// If ANSI 256 color support is unavailable, this will automatically convert the color
// to the closest available color.
func (builder *Builder) Ansi256(code uint8) func(strs ...string) string {
	return builder.WithAnsi256(code).applyStyle
}

// WithAnsi256 returns a Builder that generates strings with the specified color.
// Note that if ANSI 256 support is unavailable, this will automatically
// convert the color to the closest available color.
func (builder *Builder) WithAnsi256(code uint8) *Builder {
	if builder.shared.Level < LevelAnsi256 {
		ansiCode := ansistyles.Ansi256ToAnsi(code)
		return createBuilder(builder, ansistyles.Ansi(ansiCode), ansistyles.Close)
	}
	return createBuilder(builder, ansistyles.Ansi256(code), ansistyles.Close)
}

// BgAnsi256 returns a function which colors the background of a string using the ANSI 256 color pallette.
// If ANSI 256 color support is unavailable, this will automatically convert the color
// to the closest available color.
func BgAnsi256(code uint8) func(strs ...string) string {
	return rootBuilder.WithBgAnsi256(code).applyStyle
}

// WithBgAnsi256 returns a Builder that generates strings with the specified background color.
// Note that if ANSI 256 support is unavailable, this will automatically
// convert the color to the closest available color.
func WithBgAnsi256(code uint8) *Builder {
	return rootBuilder.WithBgAnsi256(code)
}

// BgAnsi256 returns a function which colors the background of a string using the ANSI 256 color pallette.
// If ANSI 256 color support is unavailable, this will automatically convert the color
// to the closest available color.
func (builder *Builder) BgAnsi256(code uint8) func(strs ...string) string {
	return builder.WithBgAnsi256(code).applyStyle
}

// WithBgAnsi256 returns a Builder that generates strings with the specified background color.
// Note that if ANSI 256 support is unavailable, this will automatically
// convert the color to the closest available color.
func (builder *Builder) WithBgAnsi256(code uint8) *Builder {
	if builder.shared.Level < LevelAnsi256 {
		ansiCode := ansistyles.Ansi256ToAnsi(code)
		return createBuilder(builder, ansistyles.BgAnsi(ansiCode), ansistyles.BgClose)
	}
	return createBuilder(builder, ansistyles.BgAnsi256(code), ansistyles.BgClose)
}

// RGB returns a function which colors a string using true color support.  Note that
// if true color support is unavailable, this will automatically convert the color
// to the closest available color.
func RGB(r uint8, g uint8, b uint8) func(strs ...string) string {
	return rootBuilder.WithRGB(r, g, b).applyStyle
}

// WithRGB returns a Builder that generates strings with the specified color.
// Note that if true color support is unavailable, this will automatically
// convert the color to the closest available color.
func WithRGB(r uint8, g uint8, b uint8) *Builder {
	return rootBuilder.WithRGB(r, g, b)
}

// RGB returns a function which colors a string using true color support.  Note that
// if true color support is unavailable, this will automatically convert the color
// to the closest available color.
func (builder *Builder) RGB(r uint8, g uint8, b uint8) func(strs ...string) string {
	return builder.WithRGB(r, g, b).applyStyle
}

// WithRGB returns a Builder that generates strings with the specified color.
// Note that if true color support is unavailable, this will automatically
// convert the color to the closest available color.
func (builder *Builder) WithRGB(r uint8, g uint8, b uint8) *Builder {
	if builder.shared.Level < LevelAnsi16m {
		return builder.WithAnsi256(ansistyles.RGBToAnsi256(r, g, b))
	}
	return createBuilder(builder, ansistyles.Ansi16m(r, g, b), ansistyles.Close)
}

// BgRGB returns a function which colors the background of a string using true color support.  Note that
// if true color support is unavailable, this will automatically convert the color
// to the closest available color.
func BgRGB(r uint8, g uint8, b uint8) func(strs ...string) string {
	return rootBuilder.WithBgRGB(r, g, b).applyStyle
}

// WithBgRGB returns a Builder that generates strings with the specified background color.
// Note that if true color support is unavailable, this will automatically
// convert the color to the closest available color.
func WithBgRGB(r uint8, g uint8, b uint8) *Builder {
	return rootBuilder.WithBgRGB(r, g, b)
}

// BgRGB returns a function which colors the background of a string using true color support.  Note that
// if true color support is unavailable, this will automatically convert the color
// to the closest available color.
func (builder *Builder) BgRGB(r uint8, g uint8, b uint8) func(strs ...string) string {
	return builder.WithBgRGB(r, g, b).applyStyle
}

// WithBgRGB returns a Builder that generates strings with the specified background color.
// Note that if true color support is unavailable, this will automatically
// convert the color to the closest available color.
func (builder *Builder) WithBgRGB(r uint8, g uint8, b uint8) *Builder {
	if builder.shared.Level < LevelAnsi16m {
		return builder.WithBgAnsi256(ansistyles.RGBToAnsi256(r, g, b))
	}
	return createBuilder(builder, ansistyles.BgAnsi16m(r, g, b), ansistyles.BgClose)
}

// Hex returns a function which colors a string using true color support, from a
// hexadecimal color string (e.g. "#FF00FF" or "#FFF"). If true color support is
// unavailable, this will automatically convert the color  to the closest
// available color.
func Hex(hex string) func(strs ...string) string {
	return rootBuilder.WithRGB(ansistyles.HexToRGB(hex)).applyStyle
}

// WithHex returns a Builder that generates strings with the specified color.
// Note that if true color support is unavailable, this will automatically
// convert the color to the closest available color.
func WithHex(hex string) *Builder {
	return rootBuilder.WithRGB(ansistyles.HexToRGB(hex))
}

// Hex returns a function which colors a string using true color support, from a
// hexadecimal color string (e.g. "#FF00FF" or "#FFF"). If true color support is
// unavailable, this will automatically convert the color  to the closest
// available color.
func (builder *Builder) Hex(hex string) func(strs ...string) string {
	return builder.WithRGB(ansistyles.HexToRGB(hex)).applyStyle
}

// WithHex returns a Builder that generates strings with the specified color.
// Note that if true color support is unavailable, this will automatically
// convert the color to the closest available color.
func (builder *Builder) WithHex(hex string) *Builder {
	return builder.WithRGB(ansistyles.HexToRGB(hex))
}

// BgHex returns a function which colors the background of a string using true color support, from a
// hexadecimal color string (e.g. "#FF00FF" or "#FFF"). If true color support is
// unavailable, this will automatically convert the color  to the closest
// available color.
func BgHex(hex string) func(strs ...string) string {
	return rootBuilder.WithBgRGB(ansistyles.HexToRGB(hex)).applyStyle
}

// WithBgHex returns a Builder that generates strings with the specified background color.
// Note that if true color support is unavailable, this will automatically
// convert the color to the closest available color.
func WithBgHex(hex string) *Builder {
	return rootBuilder.WithBgRGB(ansistyles.HexToRGB(hex))
}

// BgHex returns a function which colors the background of a string using true color support, from a
// hexadecimal color string (e.g. "#FF00FF" or "#FFF"). If true color support is
// unavailable, this will automatically convert the color  to the closest
// available color.
func (builder *Builder) BgHex(hex string) func(strs ...string) string {
	return builder.WithBgRGB(ansistyles.HexToRGB(hex)).applyStyle
}

// WithBgHex returns a Builder that generates strings with the specified background color.
// Note that if true color support is unavailable, this will automatically
// convert the color to the closest available color.
func (builder *Builder) WithBgHex(hex string) *Builder {
	return builder.WithBgRGB(ansistyles.HexToRGB(hex))
}
