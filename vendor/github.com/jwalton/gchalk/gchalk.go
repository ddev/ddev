// Package gchalk is terminal string styling for go done right, with full Linux, MacOS, and painless Windows 10 support.
//
// GChalk is a library heavily inspired by https://github.com/chalk/chalk, the
// popular Node.js terminal color library, and using golang ports of supports-color
// (https://github.com/jwalton/go-supportscolor) and ansi-styles
// (https://github.com/jwalton/gchalk/pkg/ansistyles).
//
// A very simple usage example would be:
//
//     fmt.Println(gchalk.Blue("This line is blue"))
//
// Note that this works on all platforms - there's no need to write to a special
// stream or use a special print function to get color on Windows 10.
//
// Some examples:
//
//     // Combine styled and normal strings
//     fmt.Println(gchalk.Blue("Hello") + " World" + gchalk.Red("!"))
//
//     // Compose multiple styles using the chainable API
//     fmt.Println(gchalk.WithBlue().WithBgRed().Bold("Hello world!"))
//
//     // Pass in multiple arguments
//     fmt.Println(gchalk.Blue("Hello", "World!", "Foo", "bar", "biz", "baz"))
//
//     // Nest styles
//     fmt.Println(gchalk.Green(
//         "I am a green line " +
//         gchalk.WithBlue().WithUnderline().Bold("with a blue substring") +
//         " that becomes green again!"
//     ))
//
//     // Use RGB colors in terminal emulators that support it.
//     fmt.Println(gchalk.WithRGB(123, 45, 67).Underline("Underlined reddish color"))
//     fmt.Println(gchalk.WithHex("#DEADED").Bold("Bold gray!"))
//
//     // Write to stderr:
//     os.Stderr.WriteString(gchalk.Stderr.Red("Ohs noes!\n"))
//
// See the README.md for more details.
//
package gchalk

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jwalton/go-supportscolor"
)

// ColorLevel represents the ANSI color level supported by the terminal.
type ColorLevel = supportscolor.ColorLevel

const (
	// LevelNone represents a terminal that does not support color at all.
	LevelNone ColorLevel = supportscolor.None
	// LevelBasic represents a terminal with basic 16 color support.
	LevelBasic ColorLevel = supportscolor.Basic
	// LevelAnsi256 represents a terminal with 256 color support.
	LevelAnsi256 ColorLevel = supportscolor.Ansi256
	// LevelAnsi16m represents a terminal with full true color support.
	LevelAnsi16m ColorLevel = supportscolor.Ansi16m
)

type stylerData struct {
	open     string
	close    string
	openAll  string
	closeAll string
	parent   *stylerData
}

// builderShared contains data shared between all builders.
type builderShared struct {
	Level ColorLevel
	mutex sync.Mutex
}

// A Builder is used to define and chain together styles.
//
// Instances of Builder cannot be constructed directly - you can build a new
// instance via the New() function, which will give you an instance you can
// configure without modifying the "default" Builder.
//
type Builder struct {
	styler *stylerData
	shared *builderShared

	bgBlack         *Builder
	bgBrightBlack   *Builder
	bgBlue          *Builder
	bgBrightBlue    *Builder
	bgCyan          *Builder
	bgBrightCyan    *Builder
	bgGray          *Builder
	bgGreen         *Builder
	bgBrightGreen   *Builder
	bgGrey          *Builder
	bgMagenta       *Builder
	bgBrightMagenta *Builder
	bgRed           *Builder
	bgBrightRed     *Builder
	bgWhite         *Builder
	bgBrightWhite   *Builder
	bgYellow        *Builder
	bgBrightYellow  *Builder
	black           *Builder
	brightBlack     *Builder
	blue            *Builder
	brightBlue      *Builder
	cyan            *Builder
	brightCyan      *Builder
	gray            *Builder
	green           *Builder
	brightGreen     *Builder
	grey            *Builder
	magenta         *Builder
	brightMagenta   *Builder
	red             *Builder
	brightRed       *Builder
	white           *Builder
	brightWhite     *Builder
	yellow          *Builder
	brightYellow    *Builder
	bold            *Builder
	dim             *Builder
	hidden          *Builder
	inverse         *Builder
	italic          *Builder
	overline        *Builder
	strikethrough   *Builder
	underline       *Builder
	reset           *Builder
}

// An Option which can be passed to `New()`.
type Option func(*Builder)

// ForceLevel is an option that can be passed to `New` to force the color level
// used.
func ForceLevel(level ColorLevel) Option {
	return func(builder *Builder) {
		builder.shared.Level = level
	}
}

// New creates a new instance of GChalk.
func New(options ...Option) *Builder {
	builder := &Builder{styler: nil}

	builder.shared = &builderShared{
		Level: supportscolor.Stdout().Level,
	}

	for index := range options {
		options[index](builder)
	}

	return builder
}

// rootBuilder is the default GChalk instance, pre-configured for stdout.
var rootBuilder = New()

// Stderr is an instance of GChalk pre-configured for stderr.  Use this when coloring
// strings you intend to write the stderr.
var Stderr = New(
	ForceLevel(ColorLevel(supportscolor.Stderr().Level)),
)

func createBuilder(parentBuilder *Builder, open string, close string) *Builder {
	var parent *stylerData
	if parentBuilder.styler != nil {
		parent = parentBuilder.styler
	}

	openAll := open
	closeAll := close
	if parent != nil {
		openAll = parent.openAll + open
		closeAll = close + parent.closeAll
	}

	return &Builder{
		shared: parentBuilder.shared,
		styler: &stylerData{
			open:     open,
			close:    close,
			openAll:  openAll,
			closeAll: closeAll,
			parent:   parent,
		},
	}
}

func (builder *Builder) applyStyle(strs ...string) string {
	if len(strs) == 0 {
		return ""
	}

	var str string
	if len(strs) == 1 {
		str = strs[0]
	} else {
		str = strings.Join(strs, " ")
	}

	if (builder.shared.Level <= LevelNone) || str == "" {
		return str
	}

	styler := builder.styler

	if styler == nil {
		return str
	}

	openAll := styler.openAll
	closeAll := styler.closeAll

	if strings.Contains(str, "\u001B") {
		for styler != nil {
			// Replace any instances already present with a re-opening code
			// otherwise only the part of the string until said closing code
			// will be colored, and the rest will simply be 'plain'.
			if styler.close == "\u001b[22m" {
				// This is kind of a weird corner case - both "bold" and "dim"
				// close with "22", but these are actually not mutually exclusive
				// styles - you can have something both bold and dim at the same
				// time (iTerm 2, for example, will render it as a dimmer color,
				// with a bold font face).  So when we nest "dim" inside "bold",
				// if we just replace the dim's close with bold's open, we'll
				// end up with something that is dim and bold at the same time.
				// The fix here is to keep the close tag.  This can lead to
				// a big chain of close tags followed immediately by open tags
				// in cases where we do a lot of nesting, and in any other
				// case this is pointless (as a string can't be both red and
				// blue at the same time, for example), so we treat this as a
				// special case.
				str = strings.ReplaceAll(str, styler.close, styler.close+styler.open)
			} else {
				str = strings.ReplaceAll(str, styler.close, styler.open)
			}

			styler = styler.parent
		}
	}

	// We can move both next actions out of loop, because remaining actions in loop won't have
	// any/visible effect on parts we add here. Close the styling before a linebreak and reopen
	// after next line to fix a bleed issue on macOS: https://github.com/chalk/chalk/pull/92
	if strings.Contains(str, "\n") {
		str = stringEncaseCRLF(str, closeAll, openAll)
	}

	// Concat using "+" instead of fmt.Sprintf, because it's about four times faster.
	return openAll + str + closeAll
}

// SetLevel is used to override the auto-detected color level.
func SetLevel(level ColorLevel) {
	rootBuilder.SetLevel(level)
}

// GetLevel returns the currently configured color level.
func GetLevel() ColorLevel {
	return rootBuilder.GetLevel()
}

// SetLevel is used to override the auto-detected color level for a builder.  Calling
// this at any level of the builder will affect the entire instance of the builder.
func (builder *Builder) SetLevel(level ColorLevel) {
	builder.shared.Level = level
}

// GetLevel returns the currently configured level for this builder.
func (builder *Builder) GetLevel() ColorLevel {
	return builder.shared.Level
}

// StyleMust will return a function which colors a string with the specified
// styles. Styles can be specified as a named style (e.g. "red", "bgRed", "bgred"),
// or as a hex color ("#ff00ff" or "bg#ff00ff").  If the style cannot
// be parsed, this will panic.
func StyleMust(styles ...string) func(strs ...string) string {
	return rootBuilder.WithStyleMust(styles...).applyStyle
}

// WithStyleMust will construct a Builder that generates strings with the specified
// styles. Styles can be specified as a named style (e.g. "red", "bgRed", "bgred"),
// or as a hex color ("#ff00ff" or "bg#ff00ff").  If the style cannot
// be parsed, this will panic.
func WithStyleMust(styles ...string) *Builder {
	return rootBuilder.WithStyleMust(styles...)
}

// StyleMust will return a function which colors a string with the specified
// styles. Styles can be specified as a named style (e.g. "red", "bgRed", "bgred"),
// or as a hex color ("#ff00ff" or "bg#ff00ff").  If the style cannot
// be parsed, this will panic.
func (builder *Builder) StyleMust(styles ...string) func(strs ...string) string {
	return builder.WithStyleMust(styles...).applyStyle
}

// WithStyleMust will construct a Builder that generates strings with the specified
// styles. Styles can be specified as a named style (e.g. "red", "bgRed", "bgred"),
// or as a hex color ("#ff00ff" or "bg#ff00ff").  If the style cannot
// be parsed, this will panic.
func (builder *Builder) WithStyleMust(styles ...string) *Builder {
	var result, err = builder.WithStyle(styles...)
	if err != nil {
		panic(err)
	}

	return result
}

// Style will return a function which colors a string with the specified
// styles. Styles can be specified as a named style (e.g. "red", "bgRed", "bgred"),
// or as a hex color ("#ff00ff" or "bg#ff00ff").  If the style cannot
// be parsed, this will return an error.
func Style(styles ...string) (func(strs ...string) string, error) {
	newBuilder, err := rootBuilder.WithStyle(styles...)
	if err != nil {
		return rootBuilder.applyStyle, err
	}
	return newBuilder.applyStyle, nil
}

// WithStyle will construct a Builder that generates strings with the specified
// styles. Styles can be specified as a named style (e.g. "red", "bgRed", "bgred"),
// or as a hex color ("#ff00ff" or "bg#ff00ff").  If the style cannot
// be parsed, this will return an error.
func WithStyle(styles ...string) (*Builder, error) {
	return rootBuilder.WithStyle(styles...)
}

// Style will return a function which colors a string with the specified
// styles. Styles can be specified as a named style (e.g. "red", "bgRed", "bgred"),
// or as a hex color ("#ff00ff" or "bg#ff00ff").  If the style cannot
// be parsed, this will return an error.
func (builder *Builder) Style(styles ...string) (func(strs ...string) string, error) {
	newBuilder, err := rootBuilder.WithStyle(styles...)
	if err != nil {
		return rootBuilder.applyStyle, err
	}
	return newBuilder.applyStyle, nil
}

// WithStyle will construct a Builder that generates strings with the specified
// styles. Styles can be specified as a named style (e.g. "red", "bgRed", "bgred"),
// or as a hex color ("#ff00ff" or "bg#ff00ff").  If the style cannot
// be parsed, this will return an error.
func (builder *Builder) WithStyle(styles ...string) (*Builder, error) {
	var err error = nil
	var result *Builder = builder

	for _, style := range styles {
		newBuilder := result.getBuilderForStyle(style)

		if newBuilder == nil {
			// Handle hex codes.
			if style[0] == '#' {
				newBuilder = builder.WithHex(style)
			} else if strings.HasPrefix(style, "bg#") {
				newBuilder = builder.WithBgHex(style[2:])
			}
		}

		if newBuilder != nil {
			result = newBuilder
		} else {
			err = fmt.Errorf("No such style: %s", style)
			result = builder
		}
	}

	return result, err
}

// Paint will apply a style to a string.
// This is similar to the `paint()` function from Rust's `ansi_term` crate.
//
//     gchalk.WithRed().Paint("Hello World!")
//
func (builder *Builder) Paint(strs ...string) string {
	return builder.applyStyle(strs...)
}

// Sprintf is a convenience function for coloring formatted strings.
//
//     gchalk.WithRed().Sprtinf("Hello %s", "World!")
//
func (builder *Builder) Sprintf(format string, a ...interface{}) string {
	return builder.applyStyle(fmt.Sprintf(format, a...))
}
