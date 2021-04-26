package gchalk

// This file was generated.  Don't edit it directly.

import (
	"github.com/jwalton/gchalk/pkg/ansistyles"
)

// Black returns a string where the color is black.
func Black(str ...string) string {
	return rootBuilder.WithBlack().applyStyle(str...)
}

// WithBlack returns a Builder that generates strings where the color is black,
// and further styles can be applied via chaining.
func WithBlack() *Builder {
	return rootBuilder.WithBlack()
}

// Black returns a string where the color is black, in addition to other styles from this builder.
func (builder *Builder) Black(str ...string) string {
	return builder.WithBlack().applyStyle(str...)
}

// WithBlack returns a Builder that generates strings where the color is black,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBlack() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.black == nil {
		builder.black = createBuilder(builder, ansistyles.Black.Open, ansistyles.Black.Close)
	}
	return builder.black
}

// Blue returns a string where the color is blue.
func Blue(str ...string) string {
	return rootBuilder.WithBlue().applyStyle(str...)
}

// WithBlue returns a Builder that generates strings where the color is blue,
// and further styles can be applied via chaining.
func WithBlue() *Builder {
	return rootBuilder.WithBlue()
}

// Blue returns a string where the color is blue, in addition to other styles from this builder.
func (builder *Builder) Blue(str ...string) string {
	return builder.WithBlue().applyStyle(str...)
}

// WithBlue returns a Builder that generates strings where the color is blue,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBlue() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.blue == nil {
		builder.blue = createBuilder(builder, ansistyles.Blue.Open, ansistyles.Blue.Close)
	}
	return builder.blue
}

// BrightBlack returns a string where the color is brightBlack.
func BrightBlack(str ...string) string {
	return rootBuilder.WithBrightBlack().applyStyle(str...)
}

// WithBrightBlack returns a Builder that generates strings where the color is brightBlack,
// and further styles can be applied via chaining.
func WithBrightBlack() *Builder {
	return rootBuilder.WithBrightBlack()
}

// BrightBlack returns a string where the color is brightBlack, in addition to other styles from this builder.
func (builder *Builder) BrightBlack(str ...string) string {
	return builder.WithBrightBlack().applyStyle(str...)
}

// WithBrightBlack returns a Builder that generates strings where the color is brightBlack,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBrightBlack() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.brightBlack == nil {
		builder.brightBlack = createBuilder(builder, ansistyles.BrightBlack.Open, ansistyles.BrightBlack.Close)
	}
	return builder.brightBlack
}

// BrightBlue returns a string where the color is brightBlue.
func BrightBlue(str ...string) string {
	return rootBuilder.WithBrightBlue().applyStyle(str...)
}

// WithBrightBlue returns a Builder that generates strings where the color is brightBlue,
// and further styles can be applied via chaining.
func WithBrightBlue() *Builder {
	return rootBuilder.WithBrightBlue()
}

// BrightBlue returns a string where the color is brightBlue, in addition to other styles from this builder.
func (builder *Builder) BrightBlue(str ...string) string {
	return builder.WithBrightBlue().applyStyle(str...)
}

// WithBrightBlue returns a Builder that generates strings where the color is brightBlue,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBrightBlue() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.brightBlue == nil {
		builder.brightBlue = createBuilder(builder, ansistyles.BrightBlue.Open, ansistyles.BrightBlue.Close)
	}
	return builder.brightBlue
}

// BrightCyan returns a string where the color is brightCyan.
func BrightCyan(str ...string) string {
	return rootBuilder.WithBrightCyan().applyStyle(str...)
}

// WithBrightCyan returns a Builder that generates strings where the color is brightCyan,
// and further styles can be applied via chaining.
func WithBrightCyan() *Builder {
	return rootBuilder.WithBrightCyan()
}

// BrightCyan returns a string where the color is brightCyan, in addition to other styles from this builder.
func (builder *Builder) BrightCyan(str ...string) string {
	return builder.WithBrightCyan().applyStyle(str...)
}

// WithBrightCyan returns a Builder that generates strings where the color is brightCyan,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBrightCyan() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.brightCyan == nil {
		builder.brightCyan = createBuilder(builder, ansistyles.BrightCyan.Open, ansistyles.BrightCyan.Close)
	}
	return builder.brightCyan
}

// BrightGreen returns a string where the color is brightGreen.
func BrightGreen(str ...string) string {
	return rootBuilder.WithBrightGreen().applyStyle(str...)
}

// WithBrightGreen returns a Builder that generates strings where the color is brightGreen,
// and further styles can be applied via chaining.
func WithBrightGreen() *Builder {
	return rootBuilder.WithBrightGreen()
}

// BrightGreen returns a string where the color is brightGreen, in addition to other styles from this builder.
func (builder *Builder) BrightGreen(str ...string) string {
	return builder.WithBrightGreen().applyStyle(str...)
}

// WithBrightGreen returns a Builder that generates strings where the color is brightGreen,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBrightGreen() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.brightGreen == nil {
		builder.brightGreen = createBuilder(builder, ansistyles.BrightGreen.Open, ansistyles.BrightGreen.Close)
	}
	return builder.brightGreen
}

// BrightMagenta returns a string where the color is brightMagenta.
func BrightMagenta(str ...string) string {
	return rootBuilder.WithBrightMagenta().applyStyle(str...)
}

// WithBrightMagenta returns a Builder that generates strings where the color is brightMagenta,
// and further styles can be applied via chaining.
func WithBrightMagenta() *Builder {
	return rootBuilder.WithBrightMagenta()
}

// BrightMagenta returns a string where the color is brightMagenta, in addition to other styles from this builder.
func (builder *Builder) BrightMagenta(str ...string) string {
	return builder.WithBrightMagenta().applyStyle(str...)
}

// WithBrightMagenta returns a Builder that generates strings where the color is brightMagenta,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBrightMagenta() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.brightMagenta == nil {
		builder.brightMagenta = createBuilder(builder, ansistyles.BrightMagenta.Open, ansistyles.BrightMagenta.Close)
	}
	return builder.brightMagenta
}

// BrightRed returns a string where the color is brightRed.
func BrightRed(str ...string) string {
	return rootBuilder.WithBrightRed().applyStyle(str...)
}

// WithBrightRed returns a Builder that generates strings where the color is brightRed,
// and further styles can be applied via chaining.
func WithBrightRed() *Builder {
	return rootBuilder.WithBrightRed()
}

// BrightRed returns a string where the color is brightRed, in addition to other styles from this builder.
func (builder *Builder) BrightRed(str ...string) string {
	return builder.WithBrightRed().applyStyle(str...)
}

// WithBrightRed returns a Builder that generates strings where the color is brightRed,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBrightRed() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.brightRed == nil {
		builder.brightRed = createBuilder(builder, ansistyles.BrightRed.Open, ansistyles.BrightRed.Close)
	}
	return builder.brightRed
}

// BrightWhite returns a string where the color is brightWhite.
func BrightWhite(str ...string) string {
	return rootBuilder.WithBrightWhite().applyStyle(str...)
}

// WithBrightWhite returns a Builder that generates strings where the color is brightWhite,
// and further styles can be applied via chaining.
func WithBrightWhite() *Builder {
	return rootBuilder.WithBrightWhite()
}

// BrightWhite returns a string where the color is brightWhite, in addition to other styles from this builder.
func (builder *Builder) BrightWhite(str ...string) string {
	return builder.WithBrightWhite().applyStyle(str...)
}

// WithBrightWhite returns a Builder that generates strings where the color is brightWhite,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBrightWhite() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.brightWhite == nil {
		builder.brightWhite = createBuilder(builder, ansistyles.BrightWhite.Open, ansistyles.BrightWhite.Close)
	}
	return builder.brightWhite
}

// BrightYellow returns a string where the color is brightYellow.
func BrightYellow(str ...string) string {
	return rootBuilder.WithBrightYellow().applyStyle(str...)
}

// WithBrightYellow returns a Builder that generates strings where the color is brightYellow,
// and further styles can be applied via chaining.
func WithBrightYellow() *Builder {
	return rootBuilder.WithBrightYellow()
}

// BrightYellow returns a string where the color is brightYellow, in addition to other styles from this builder.
func (builder *Builder) BrightYellow(str ...string) string {
	return builder.WithBrightYellow().applyStyle(str...)
}

// WithBrightYellow returns a Builder that generates strings where the color is brightYellow,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBrightYellow() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.brightYellow == nil {
		builder.brightYellow = createBuilder(builder, ansistyles.BrightYellow.Open, ansistyles.BrightYellow.Close)
	}
	return builder.brightYellow
}

// Cyan returns a string where the color is cyan.
func Cyan(str ...string) string {
	return rootBuilder.WithCyan().applyStyle(str...)
}

// WithCyan returns a Builder that generates strings where the color is cyan,
// and further styles can be applied via chaining.
func WithCyan() *Builder {
	return rootBuilder.WithCyan()
}

// Cyan returns a string where the color is cyan, in addition to other styles from this builder.
func (builder *Builder) Cyan(str ...string) string {
	return builder.WithCyan().applyStyle(str...)
}

// WithCyan returns a Builder that generates strings where the color is cyan,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithCyan() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.cyan == nil {
		builder.cyan = createBuilder(builder, ansistyles.Cyan.Open, ansistyles.Cyan.Close)
	}
	return builder.cyan
}

// Gray returns a string where the color is gray.
func Gray(str ...string) string {
	return rootBuilder.WithGray().applyStyle(str...)
}

// WithGray returns a Builder that generates strings where the color is gray,
// and further styles can be applied via chaining.
func WithGray() *Builder {
	return rootBuilder.WithGray()
}

// Gray returns a string where the color is gray, in addition to other styles from this builder.
func (builder *Builder) Gray(str ...string) string {
	return builder.WithGray().applyStyle(str...)
}

// WithGray returns a Builder that generates strings where the color is gray,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithGray() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.gray == nil {
		builder.gray = createBuilder(builder, ansistyles.Gray.Open, ansistyles.Gray.Close)
	}
	return builder.gray
}

// Green returns a string where the color is green.
func Green(str ...string) string {
	return rootBuilder.WithGreen().applyStyle(str...)
}

// WithGreen returns a Builder that generates strings where the color is green,
// and further styles can be applied via chaining.
func WithGreen() *Builder {
	return rootBuilder.WithGreen()
}

// Green returns a string where the color is green, in addition to other styles from this builder.
func (builder *Builder) Green(str ...string) string {
	return builder.WithGreen().applyStyle(str...)
}

// WithGreen returns a Builder that generates strings where the color is green,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithGreen() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.green == nil {
		builder.green = createBuilder(builder, ansistyles.Green.Open, ansistyles.Green.Close)
	}
	return builder.green
}

// Grey returns a string where the color is grey.
func Grey(str ...string) string {
	return rootBuilder.WithGrey().applyStyle(str...)
}

// WithGrey returns a Builder that generates strings where the color is grey,
// and further styles can be applied via chaining.
func WithGrey() *Builder {
	return rootBuilder.WithGrey()
}

// Grey returns a string where the color is grey, in addition to other styles from this builder.
func (builder *Builder) Grey(str ...string) string {
	return builder.WithGrey().applyStyle(str...)
}

// WithGrey returns a Builder that generates strings where the color is grey,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithGrey() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.grey == nil {
		builder.grey = createBuilder(builder, ansistyles.Grey.Open, ansistyles.Grey.Close)
	}
	return builder.grey
}

// Magenta returns a string where the color is magenta.
func Magenta(str ...string) string {
	return rootBuilder.WithMagenta().applyStyle(str...)
}

// WithMagenta returns a Builder that generates strings where the color is magenta,
// and further styles can be applied via chaining.
func WithMagenta() *Builder {
	return rootBuilder.WithMagenta()
}

// Magenta returns a string where the color is magenta, in addition to other styles from this builder.
func (builder *Builder) Magenta(str ...string) string {
	return builder.WithMagenta().applyStyle(str...)
}

// WithMagenta returns a Builder that generates strings where the color is magenta,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithMagenta() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.magenta == nil {
		builder.magenta = createBuilder(builder, ansistyles.Magenta.Open, ansistyles.Magenta.Close)
	}
	return builder.magenta
}

// Red returns a string where the color is red.
func Red(str ...string) string {
	return rootBuilder.WithRed().applyStyle(str...)
}

// WithRed returns a Builder that generates strings where the color is red,
// and further styles can be applied via chaining.
func WithRed() *Builder {
	return rootBuilder.WithRed()
}

// Red returns a string where the color is red, in addition to other styles from this builder.
func (builder *Builder) Red(str ...string) string {
	return builder.WithRed().applyStyle(str...)
}

// WithRed returns a Builder that generates strings where the color is red,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithRed() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.red == nil {
		builder.red = createBuilder(builder, ansistyles.Red.Open, ansistyles.Red.Close)
	}
	return builder.red
}

// White returns a string where the color is white.
func White(str ...string) string {
	return rootBuilder.WithWhite().applyStyle(str...)
}

// WithWhite returns a Builder that generates strings where the color is white,
// and further styles can be applied via chaining.
func WithWhite() *Builder {
	return rootBuilder.WithWhite()
}

// White returns a string where the color is white, in addition to other styles from this builder.
func (builder *Builder) White(str ...string) string {
	return builder.WithWhite().applyStyle(str...)
}

// WithWhite returns a Builder that generates strings where the color is white,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithWhite() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.white == nil {
		builder.white = createBuilder(builder, ansistyles.White.Open, ansistyles.White.Close)
	}
	return builder.white
}

// Yellow returns a string where the color is yellow.
func Yellow(str ...string) string {
	return rootBuilder.WithYellow().applyStyle(str...)
}

// WithYellow returns a Builder that generates strings where the color is yellow,
// and further styles can be applied via chaining.
func WithYellow() *Builder {
	return rootBuilder.WithYellow()
}

// Yellow returns a string where the color is yellow, in addition to other styles from this builder.
func (builder *Builder) Yellow(str ...string) string {
	return builder.WithYellow().applyStyle(str...)
}

// WithYellow returns a Builder that generates strings where the color is yellow,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithYellow() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.yellow == nil {
		builder.yellow = createBuilder(builder, ansistyles.Yellow.Open, ansistyles.Yellow.Close)
	}
	return builder.yellow
}

// BgBlack returns a string where the background color is Black.
func BgBlack(str ...string) string {
	return rootBuilder.WithBgBlack().applyStyle(str...)
}

// WithBgBlack returns a Builder that generates strings where the background color is Black,
// and further styles can be applied via chaining.
func WithBgBlack() *Builder {
	return rootBuilder.WithBgBlack()
}

// BgBlack returns a string where the background color is Black, in addition to other styles from this builder.
func (builder *Builder) BgBlack(str ...string) string {
	return builder.WithBgBlack().applyStyle(str...)
}

// WithBgBlack returns a Builder that generates strings where the background color is Black,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgBlack() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgBlack == nil {
		builder.bgBlack = createBuilder(builder, ansistyles.BgBlack.Open, ansistyles.BgBlack.Close)
	}
	return builder.bgBlack
}

// BgBlue returns a string where the background color is Blue.
func BgBlue(str ...string) string {
	return rootBuilder.WithBgBlue().applyStyle(str...)
}

// WithBgBlue returns a Builder that generates strings where the background color is Blue,
// and further styles can be applied via chaining.
func WithBgBlue() *Builder {
	return rootBuilder.WithBgBlue()
}

// BgBlue returns a string where the background color is Blue, in addition to other styles from this builder.
func (builder *Builder) BgBlue(str ...string) string {
	return builder.WithBgBlue().applyStyle(str...)
}

// WithBgBlue returns a Builder that generates strings where the background color is Blue,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgBlue() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgBlue == nil {
		builder.bgBlue = createBuilder(builder, ansistyles.BgBlue.Open, ansistyles.BgBlue.Close)
	}
	return builder.bgBlue
}

// BgBrightBlack returns a string where the background color is BrightBlack.
func BgBrightBlack(str ...string) string {
	return rootBuilder.WithBgBrightBlack().applyStyle(str...)
}

// WithBgBrightBlack returns a Builder that generates strings where the background color is BrightBlack,
// and further styles can be applied via chaining.
func WithBgBrightBlack() *Builder {
	return rootBuilder.WithBgBrightBlack()
}

// BgBrightBlack returns a string where the background color is BrightBlack, in addition to other styles from this builder.
func (builder *Builder) BgBrightBlack(str ...string) string {
	return builder.WithBgBrightBlack().applyStyle(str...)
}

// WithBgBrightBlack returns a Builder that generates strings where the background color is BrightBlack,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgBrightBlack() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgBrightBlack == nil {
		builder.bgBrightBlack = createBuilder(builder, ansistyles.BgBrightBlack.Open, ansistyles.BgBrightBlack.Close)
	}
	return builder.bgBrightBlack
}

// BgBrightBlue returns a string where the background color is BrightBlue.
func BgBrightBlue(str ...string) string {
	return rootBuilder.WithBgBrightBlue().applyStyle(str...)
}

// WithBgBrightBlue returns a Builder that generates strings where the background color is BrightBlue,
// and further styles can be applied via chaining.
func WithBgBrightBlue() *Builder {
	return rootBuilder.WithBgBrightBlue()
}

// BgBrightBlue returns a string where the background color is BrightBlue, in addition to other styles from this builder.
func (builder *Builder) BgBrightBlue(str ...string) string {
	return builder.WithBgBrightBlue().applyStyle(str...)
}

// WithBgBrightBlue returns a Builder that generates strings where the background color is BrightBlue,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgBrightBlue() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgBrightBlue == nil {
		builder.bgBrightBlue = createBuilder(builder, ansistyles.BgBrightBlue.Open, ansistyles.BgBrightBlue.Close)
	}
	return builder.bgBrightBlue
}

// BgBrightCyan returns a string where the background color is BrightCyan.
func BgBrightCyan(str ...string) string {
	return rootBuilder.WithBgBrightCyan().applyStyle(str...)
}

// WithBgBrightCyan returns a Builder that generates strings where the background color is BrightCyan,
// and further styles can be applied via chaining.
func WithBgBrightCyan() *Builder {
	return rootBuilder.WithBgBrightCyan()
}

// BgBrightCyan returns a string where the background color is BrightCyan, in addition to other styles from this builder.
func (builder *Builder) BgBrightCyan(str ...string) string {
	return builder.WithBgBrightCyan().applyStyle(str...)
}

// WithBgBrightCyan returns a Builder that generates strings where the background color is BrightCyan,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgBrightCyan() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgBrightCyan == nil {
		builder.bgBrightCyan = createBuilder(builder, ansistyles.BgBrightCyan.Open, ansistyles.BgBrightCyan.Close)
	}
	return builder.bgBrightCyan
}

// BgBrightGreen returns a string where the background color is BrightGreen.
func BgBrightGreen(str ...string) string {
	return rootBuilder.WithBgBrightGreen().applyStyle(str...)
}

// WithBgBrightGreen returns a Builder that generates strings where the background color is BrightGreen,
// and further styles can be applied via chaining.
func WithBgBrightGreen() *Builder {
	return rootBuilder.WithBgBrightGreen()
}

// BgBrightGreen returns a string where the background color is BrightGreen, in addition to other styles from this builder.
func (builder *Builder) BgBrightGreen(str ...string) string {
	return builder.WithBgBrightGreen().applyStyle(str...)
}

// WithBgBrightGreen returns a Builder that generates strings where the background color is BrightGreen,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgBrightGreen() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgBrightGreen == nil {
		builder.bgBrightGreen = createBuilder(builder, ansistyles.BgBrightGreen.Open, ansistyles.BgBrightGreen.Close)
	}
	return builder.bgBrightGreen
}

// BgBrightMagenta returns a string where the background color is BrightMagenta.
func BgBrightMagenta(str ...string) string {
	return rootBuilder.WithBgBrightMagenta().applyStyle(str...)
}

// WithBgBrightMagenta returns a Builder that generates strings where the background color is BrightMagenta,
// and further styles can be applied via chaining.
func WithBgBrightMagenta() *Builder {
	return rootBuilder.WithBgBrightMagenta()
}

// BgBrightMagenta returns a string where the background color is BrightMagenta, in addition to other styles from this builder.
func (builder *Builder) BgBrightMagenta(str ...string) string {
	return builder.WithBgBrightMagenta().applyStyle(str...)
}

// WithBgBrightMagenta returns a Builder that generates strings where the background color is BrightMagenta,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgBrightMagenta() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgBrightMagenta == nil {
		builder.bgBrightMagenta = createBuilder(builder, ansistyles.BgBrightMagenta.Open, ansistyles.BgBrightMagenta.Close)
	}
	return builder.bgBrightMagenta
}

// BgBrightRed returns a string where the background color is BrightRed.
func BgBrightRed(str ...string) string {
	return rootBuilder.WithBgBrightRed().applyStyle(str...)
}

// WithBgBrightRed returns a Builder that generates strings where the background color is BrightRed,
// and further styles can be applied via chaining.
func WithBgBrightRed() *Builder {
	return rootBuilder.WithBgBrightRed()
}

// BgBrightRed returns a string where the background color is BrightRed, in addition to other styles from this builder.
func (builder *Builder) BgBrightRed(str ...string) string {
	return builder.WithBgBrightRed().applyStyle(str...)
}

// WithBgBrightRed returns a Builder that generates strings where the background color is BrightRed,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgBrightRed() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgBrightRed == nil {
		builder.bgBrightRed = createBuilder(builder, ansistyles.BgBrightRed.Open, ansistyles.BgBrightRed.Close)
	}
	return builder.bgBrightRed
}

// BgBrightWhite returns a string where the background color is BrightWhite.
func BgBrightWhite(str ...string) string {
	return rootBuilder.WithBgBrightWhite().applyStyle(str...)
}

// WithBgBrightWhite returns a Builder that generates strings where the background color is BrightWhite,
// and further styles can be applied via chaining.
func WithBgBrightWhite() *Builder {
	return rootBuilder.WithBgBrightWhite()
}

// BgBrightWhite returns a string where the background color is BrightWhite, in addition to other styles from this builder.
func (builder *Builder) BgBrightWhite(str ...string) string {
	return builder.WithBgBrightWhite().applyStyle(str...)
}

// WithBgBrightWhite returns a Builder that generates strings where the background color is BrightWhite,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgBrightWhite() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgBrightWhite == nil {
		builder.bgBrightWhite = createBuilder(builder, ansistyles.BgBrightWhite.Open, ansistyles.BgBrightWhite.Close)
	}
	return builder.bgBrightWhite
}

// BgBrightYellow returns a string where the background color is BrightYellow.
func BgBrightYellow(str ...string) string {
	return rootBuilder.WithBgBrightYellow().applyStyle(str...)
}

// WithBgBrightYellow returns a Builder that generates strings where the background color is BrightYellow,
// and further styles can be applied via chaining.
func WithBgBrightYellow() *Builder {
	return rootBuilder.WithBgBrightYellow()
}

// BgBrightYellow returns a string where the background color is BrightYellow, in addition to other styles from this builder.
func (builder *Builder) BgBrightYellow(str ...string) string {
	return builder.WithBgBrightYellow().applyStyle(str...)
}

// WithBgBrightYellow returns a Builder that generates strings where the background color is BrightYellow,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgBrightYellow() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgBrightYellow == nil {
		builder.bgBrightYellow = createBuilder(builder, ansistyles.BgBrightYellow.Open, ansistyles.BgBrightYellow.Close)
	}
	return builder.bgBrightYellow
}

// BgCyan returns a string where the background color is Cyan.
func BgCyan(str ...string) string {
	return rootBuilder.WithBgCyan().applyStyle(str...)
}

// WithBgCyan returns a Builder that generates strings where the background color is Cyan,
// and further styles can be applied via chaining.
func WithBgCyan() *Builder {
	return rootBuilder.WithBgCyan()
}

// BgCyan returns a string where the background color is Cyan, in addition to other styles from this builder.
func (builder *Builder) BgCyan(str ...string) string {
	return builder.WithBgCyan().applyStyle(str...)
}

// WithBgCyan returns a Builder that generates strings where the background color is Cyan,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgCyan() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgCyan == nil {
		builder.bgCyan = createBuilder(builder, ansistyles.BgCyan.Open, ansistyles.BgCyan.Close)
	}
	return builder.bgCyan
}

// BgGray returns a string where the background color is Gray.
func BgGray(str ...string) string {
	return rootBuilder.WithBgGray().applyStyle(str...)
}

// WithBgGray returns a Builder that generates strings where the background color is Gray,
// and further styles can be applied via chaining.
func WithBgGray() *Builder {
	return rootBuilder.WithBgGray()
}

// BgGray returns a string where the background color is Gray, in addition to other styles from this builder.
func (builder *Builder) BgGray(str ...string) string {
	return builder.WithBgGray().applyStyle(str...)
}

// WithBgGray returns a Builder that generates strings where the background color is Gray,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgGray() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgGray == nil {
		builder.bgGray = createBuilder(builder, ansistyles.BgGray.Open, ansistyles.BgGray.Close)
	}
	return builder.bgGray
}

// BgGreen returns a string where the background color is Green.
func BgGreen(str ...string) string {
	return rootBuilder.WithBgGreen().applyStyle(str...)
}

// WithBgGreen returns a Builder that generates strings where the background color is Green,
// and further styles can be applied via chaining.
func WithBgGreen() *Builder {
	return rootBuilder.WithBgGreen()
}

// BgGreen returns a string where the background color is Green, in addition to other styles from this builder.
func (builder *Builder) BgGreen(str ...string) string {
	return builder.WithBgGreen().applyStyle(str...)
}

// WithBgGreen returns a Builder that generates strings where the background color is Green,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgGreen() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgGreen == nil {
		builder.bgGreen = createBuilder(builder, ansistyles.BgGreen.Open, ansistyles.BgGreen.Close)
	}
	return builder.bgGreen
}

// BgGrey returns a string where the background color is Grey.
func BgGrey(str ...string) string {
	return rootBuilder.WithBgGrey().applyStyle(str...)
}

// WithBgGrey returns a Builder that generates strings where the background color is Grey,
// and further styles can be applied via chaining.
func WithBgGrey() *Builder {
	return rootBuilder.WithBgGrey()
}

// BgGrey returns a string where the background color is Grey, in addition to other styles from this builder.
func (builder *Builder) BgGrey(str ...string) string {
	return builder.WithBgGrey().applyStyle(str...)
}

// WithBgGrey returns a Builder that generates strings where the background color is Grey,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgGrey() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgGrey == nil {
		builder.bgGrey = createBuilder(builder, ansistyles.BgGrey.Open, ansistyles.BgGrey.Close)
	}
	return builder.bgGrey
}

// BgMagenta returns a string where the background color is Magenta.
func BgMagenta(str ...string) string {
	return rootBuilder.WithBgMagenta().applyStyle(str...)
}

// WithBgMagenta returns a Builder that generates strings where the background color is Magenta,
// and further styles can be applied via chaining.
func WithBgMagenta() *Builder {
	return rootBuilder.WithBgMagenta()
}

// BgMagenta returns a string where the background color is Magenta, in addition to other styles from this builder.
func (builder *Builder) BgMagenta(str ...string) string {
	return builder.WithBgMagenta().applyStyle(str...)
}

// WithBgMagenta returns a Builder that generates strings where the background color is Magenta,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgMagenta() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgMagenta == nil {
		builder.bgMagenta = createBuilder(builder, ansistyles.BgMagenta.Open, ansistyles.BgMagenta.Close)
	}
	return builder.bgMagenta
}

// BgRed returns a string where the background color is Red.
func BgRed(str ...string) string {
	return rootBuilder.WithBgRed().applyStyle(str...)
}

// WithBgRed returns a Builder that generates strings where the background color is Red,
// and further styles can be applied via chaining.
func WithBgRed() *Builder {
	return rootBuilder.WithBgRed()
}

// BgRed returns a string where the background color is Red, in addition to other styles from this builder.
func (builder *Builder) BgRed(str ...string) string {
	return builder.WithBgRed().applyStyle(str...)
}

// WithBgRed returns a Builder that generates strings where the background color is Red,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgRed() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgRed == nil {
		builder.bgRed = createBuilder(builder, ansistyles.BgRed.Open, ansistyles.BgRed.Close)
	}
	return builder.bgRed
}

// BgWhite returns a string where the background color is White.
func BgWhite(str ...string) string {
	return rootBuilder.WithBgWhite().applyStyle(str...)
}

// WithBgWhite returns a Builder that generates strings where the background color is White,
// and further styles can be applied via chaining.
func WithBgWhite() *Builder {
	return rootBuilder.WithBgWhite()
}

// BgWhite returns a string where the background color is White, in addition to other styles from this builder.
func (builder *Builder) BgWhite(str ...string) string {
	return builder.WithBgWhite().applyStyle(str...)
}

// WithBgWhite returns a Builder that generates strings where the background color is White,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgWhite() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgWhite == nil {
		builder.bgWhite = createBuilder(builder, ansistyles.BgWhite.Open, ansistyles.BgWhite.Close)
	}
	return builder.bgWhite
}

// BgYellow returns a string where the background color is Yellow.
func BgYellow(str ...string) string {
	return rootBuilder.WithBgYellow().applyStyle(str...)
}

// WithBgYellow returns a Builder that generates strings where the background color is Yellow,
// and further styles can be applied via chaining.
func WithBgYellow() *Builder {
	return rootBuilder.WithBgYellow()
}

// BgYellow returns a string where the background color is Yellow, in addition to other styles from this builder.
func (builder *Builder) BgYellow(str ...string) string {
	return builder.WithBgYellow().applyStyle(str...)
}

// WithBgYellow returns a Builder that generates strings where the background color is Yellow,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBgYellow() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bgYellow == nil {
		builder.bgYellow = createBuilder(builder, ansistyles.BgYellow.Open, ansistyles.BgYellow.Close)
	}
	return builder.bgYellow
}

// Bold returns a string with the bold modifier.
func Bold(str ...string) string {
	return rootBuilder.WithBold().applyStyle(str...)
}

// WithBold returns a Builder that generates strings with the bold modifier,
// and further styles can be applied via chaining.
func WithBold() *Builder {
	return rootBuilder.WithBold()
}

// Bold returns a string with the bold modifier, in addition to other styles from this builder.
func (builder *Builder) Bold(str ...string) string {
	return builder.WithBold().applyStyle(str...)
}

// WithBold returns a Builder that generates strings with the bold modifier,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithBold() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.bold == nil {
		builder.bold = createBuilder(builder, ansistyles.Bold.Open, ansistyles.Bold.Close)
	}
	return builder.bold
}

// Dim returns a string with the dim modifier.
func Dim(str ...string) string {
	return rootBuilder.WithDim().applyStyle(str...)
}

// WithDim returns a Builder that generates strings with the dim modifier,
// and further styles can be applied via chaining.
func WithDim() *Builder {
	return rootBuilder.WithDim()
}

// Dim returns a string with the dim modifier, in addition to other styles from this builder.
func (builder *Builder) Dim(str ...string) string {
	return builder.WithDim().applyStyle(str...)
}

// WithDim returns a Builder that generates strings with the dim modifier,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithDim() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.dim == nil {
		builder.dim = createBuilder(builder, ansistyles.Dim.Open, ansistyles.Dim.Close)
	}
	return builder.dim
}

// Hidden returns a string with the hidden modifier.
func Hidden(str ...string) string {
	return rootBuilder.WithHidden().applyStyle(str...)
}

// WithHidden returns a Builder that generates strings with the hidden modifier,
// and further styles can be applied via chaining.
func WithHidden() *Builder {
	return rootBuilder.WithHidden()
}

// Hidden returns a string with the hidden modifier, in addition to other styles from this builder.
func (builder *Builder) Hidden(str ...string) string {
	return builder.WithHidden().applyStyle(str...)
}

// WithHidden returns a Builder that generates strings with the hidden modifier,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithHidden() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.hidden == nil {
		builder.hidden = createBuilder(builder, ansistyles.Hidden.Open, ansistyles.Hidden.Close)
	}
	return builder.hidden
}

// Inverse returns a string with the inverse modifier.
func Inverse(str ...string) string {
	return rootBuilder.WithInverse().applyStyle(str...)
}

// WithInverse returns a Builder that generates strings with the inverse modifier,
// and further styles can be applied via chaining.
func WithInverse() *Builder {
	return rootBuilder.WithInverse()
}

// Inverse returns a string with the inverse modifier, in addition to other styles from this builder.
func (builder *Builder) Inverse(str ...string) string {
	return builder.WithInverse().applyStyle(str...)
}

// WithInverse returns a Builder that generates strings with the inverse modifier,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithInverse() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.inverse == nil {
		builder.inverse = createBuilder(builder, ansistyles.Inverse.Open, ansistyles.Inverse.Close)
	}
	return builder.inverse
}

// Italic returns a string with the italic modifier.
func Italic(str ...string) string {
	return rootBuilder.WithItalic().applyStyle(str...)
}

// WithItalic returns a Builder that generates strings with the italic modifier,
// and further styles can be applied via chaining.
func WithItalic() *Builder {
	return rootBuilder.WithItalic()
}

// Italic returns a string with the italic modifier, in addition to other styles from this builder.
func (builder *Builder) Italic(str ...string) string {
	return builder.WithItalic().applyStyle(str...)
}

// WithItalic returns a Builder that generates strings with the italic modifier,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithItalic() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.italic == nil {
		builder.italic = createBuilder(builder, ansistyles.Italic.Open, ansistyles.Italic.Close)
	}
	return builder.italic
}

// Overline returns a string with the overline modifier.
func Overline(str ...string) string {
	return rootBuilder.WithOverline().applyStyle(str...)
}

// WithOverline returns a Builder that generates strings with the overline modifier,
// and further styles can be applied via chaining.
func WithOverline() *Builder {
	return rootBuilder.WithOverline()
}

// Overline returns a string with the overline modifier, in addition to other styles from this builder.
func (builder *Builder) Overline(str ...string) string {
	return builder.WithOverline().applyStyle(str...)
}

// WithOverline returns a Builder that generates strings with the overline modifier,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithOverline() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.overline == nil {
		builder.overline = createBuilder(builder, ansistyles.Overline.Open, ansistyles.Overline.Close)
	}
	return builder.overline
}

// Reset returns a string with the reset modifier.
func Reset(str ...string) string {
	return rootBuilder.WithReset().applyStyle(str...)
}

// WithReset returns a Builder that generates strings with the reset modifier,
// and further styles can be applied via chaining.
func WithReset() *Builder {
	return rootBuilder.WithReset()
}

// Reset returns a string with the reset modifier, in addition to other styles from this builder.
func (builder *Builder) Reset(str ...string) string {
	return builder.WithReset().applyStyle(str...)
}

// WithReset returns a Builder that generates strings with the reset modifier,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithReset() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.reset == nil {
		builder.reset = createBuilder(builder, ansistyles.Reset.Open, ansistyles.Reset.Close)
	}
	return builder.reset
}

// Strikethrough returns a string with the strikethrough modifier.
func Strikethrough(str ...string) string {
	return rootBuilder.WithStrikethrough().applyStyle(str...)
}

// WithStrikethrough returns a Builder that generates strings with the strikethrough modifier,
// and further styles can be applied via chaining.
func WithStrikethrough() *Builder {
	return rootBuilder.WithStrikethrough()
}

// Strikethrough returns a string with the strikethrough modifier, in addition to other styles from this builder.
func (builder *Builder) Strikethrough(str ...string) string {
	return builder.WithStrikethrough().applyStyle(str...)
}

// WithStrikethrough returns a Builder that generates strings with the strikethrough modifier,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithStrikethrough() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.strikethrough == nil {
		builder.strikethrough = createBuilder(builder, ansistyles.Strikethrough.Open, ansistyles.Strikethrough.Close)
	}
	return builder.strikethrough
}

// Underline returns a string with the underline modifier.
func Underline(str ...string) string {
	return rootBuilder.WithUnderline().applyStyle(str...)
}

// WithUnderline returns a Builder that generates strings with the underline modifier,
// and further styles can be applied via chaining.
func WithUnderline() *Builder {
	return rootBuilder.WithUnderline()
}

// Underline returns a string with the underline modifier, in addition to other styles from this builder.
func (builder *Builder) Underline(str ...string) string {
	return builder.WithUnderline().applyStyle(str...)
}

// WithUnderline returns a Builder that generates strings with the underline modifier,
// in addition to other styles from this builder, and further styles can be applied via chaining.
func (builder *Builder) WithUnderline() *Builder {
	builder.shared.mutex.Lock()
	defer builder.shared.mutex.Unlock()

	if builder.underline == nil {
		builder.underline = createBuilder(builder, ansistyles.Underline.Open, ansistyles.Underline.Close)
	}
	return builder.underline
}
func (builder *Builder) getBuilderForStyle(style string) *Builder {
	switch {
	case strEqualsIgnoreCase(style, "black"):
		return builder.WithBlack()
	case strEqualsIgnoreCase(style, "blue"):
		return builder.WithBlue()
	case strEqualsIgnoreCase(style, "brightBlack"):
		return builder.WithBrightBlack()
	case strEqualsIgnoreCase(style, "brightBlue"):
		return builder.WithBrightBlue()
	case strEqualsIgnoreCase(style, "brightCyan"):
		return builder.WithBrightCyan()
	case strEqualsIgnoreCase(style, "brightGreen"):
		return builder.WithBrightGreen()
	case strEqualsIgnoreCase(style, "brightMagenta"):
		return builder.WithBrightMagenta()
	case strEqualsIgnoreCase(style, "brightRed"):
		return builder.WithBrightRed()
	case strEqualsIgnoreCase(style, "brightWhite"):
		return builder.WithBrightWhite()
	case strEqualsIgnoreCase(style, "brightYellow"):
		return builder.WithBrightYellow()
	case strEqualsIgnoreCase(style, "cyan"):
		return builder.WithCyan()
	case strEqualsIgnoreCase(style, "gray"):
		return builder.WithGray()
	case strEqualsIgnoreCase(style, "green"):
		return builder.WithGreen()
	case strEqualsIgnoreCase(style, "grey"):
		return builder.WithGrey()
	case strEqualsIgnoreCase(style, "magenta"):
		return builder.WithMagenta()
	case strEqualsIgnoreCase(style, "red"):
		return builder.WithRed()
	case strEqualsIgnoreCase(style, "white"):
		return builder.WithWhite()
	case strEqualsIgnoreCase(style, "yellow"):
		return builder.WithYellow()
	case strEqualsIgnoreCase(style, "bgBlack"):
		return builder.WithBgBlack()
	case strEqualsIgnoreCase(style, "bgBlue"):
		return builder.WithBgBlue()
	case strEqualsIgnoreCase(style, "bgBrightBlack"):
		return builder.WithBgBrightBlack()
	case strEqualsIgnoreCase(style, "bgBrightBlue"):
		return builder.WithBgBrightBlue()
	case strEqualsIgnoreCase(style, "bgBrightCyan"):
		return builder.WithBgBrightCyan()
	case strEqualsIgnoreCase(style, "bgBrightGreen"):
		return builder.WithBgBrightGreen()
	case strEqualsIgnoreCase(style, "bgBrightMagenta"):
		return builder.WithBgBrightMagenta()
	case strEqualsIgnoreCase(style, "bgBrightRed"):
		return builder.WithBgBrightRed()
	case strEqualsIgnoreCase(style, "bgBrightWhite"):
		return builder.WithBgBrightWhite()
	case strEqualsIgnoreCase(style, "bgBrightYellow"):
		return builder.WithBgBrightYellow()
	case strEqualsIgnoreCase(style, "bgCyan"):
		return builder.WithBgCyan()
	case strEqualsIgnoreCase(style, "bgGray"):
		return builder.WithBgGray()
	case strEqualsIgnoreCase(style, "bgGreen"):
		return builder.WithBgGreen()
	case strEqualsIgnoreCase(style, "bgGrey"):
		return builder.WithBgGrey()
	case strEqualsIgnoreCase(style, "bgMagenta"):
		return builder.WithBgMagenta()
	case strEqualsIgnoreCase(style, "bgRed"):
		return builder.WithBgRed()
	case strEqualsIgnoreCase(style, "bgWhite"):
		return builder.WithBgWhite()
	case strEqualsIgnoreCase(style, "bgYellow"):
		return builder.WithBgYellow()
	case strEqualsIgnoreCase(style, "bold"):
		return builder.WithBold()
	case strEqualsIgnoreCase(style, "dim"):
		return builder.WithDim()
	case strEqualsIgnoreCase(style, "hidden"):
		return builder.WithHidden()
	case strEqualsIgnoreCase(style, "inverse"):
		return builder.WithInverse()
	case strEqualsIgnoreCase(style, "italic"):
		return builder.WithItalic()
	case strEqualsIgnoreCase(style, "overline"):
		return builder.WithOverline()
	case strEqualsIgnoreCase(style, "reset"):
		return builder.WithReset()
	case strEqualsIgnoreCase(style, "strikethrough"):
		return builder.WithStrikethrough()
	case strEqualsIgnoreCase(style, "underline"):
		return builder.WithUnderline()
	default:
		return nil
	}
}
