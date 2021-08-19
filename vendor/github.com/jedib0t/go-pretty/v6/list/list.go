package list

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	// DefaultHTMLCSSClass stores the css-class to use when none-provided via
	// SetHTMLCSSClass(cssClass string).
	DefaultHTMLCSSClass = "go-pretty-table"
)

// listItem represents one line in the List
type listItem struct {
	Level int
	Text  string
}

// List helps print a 2-dimensional array in a human readable pretty-List.
type List struct {
	// approxSize stores the approximate output length/size
	approxSize int
	// htmlCSSClass stores the HTML CSS Class to use on the <ul> node
	htmlCSSClass string
	// items contains the list of items to render
	items []*listItem
	// level stores the current indentation level
	level int
	// outputMirror stores an io.Writer where the "Render" functions would write
	outputMirror io.Writer
	// style contains all the strings used to draw the List, and more
	style *Style
}

// AppendItem appends the item to the List of items to render.
func (l *List) AppendItem(item interface{}) {
	l.items = append(l.items, l.analyzeAndStringify(item))
}

// AppendItems appends the items to the List of items to render.
func (l *List) AppendItems(items []interface{}) {
	for _, item := range items {
		l.AppendItem(item)
	}
}

// Indent indents the following items to appear right-shifted.
func (l *List) Indent() {
	if len(l.items) == 0 {
		// should not indent when there is no item in the current level
	} else if l.level > l.items[len(l.items)-1].Level {
		// already indented compared to previous item; do not indent more
	} else {
		l.level++
	}
}

// Length returns the number of items to be rendered.
func (l *List) Length() int {
	return len(l.items)
}

// Reset sets the List to its initial state.
func (l *List) Reset() {
	l.approxSize = 0
	l.items = make([]*listItem, 0)
	l.level = 0
	l.style = nil
}

// SetHTMLCSSClass sets the the HTML CSS Class to use on the <ul> node
// when rendering the List in HTML format. Recursive lists would use a numbered
// index suffix. For ex., if the cssClass is set as "foo"; the <ul> for level 0
// would have the class set as "foo"; the <ul> for level 1 would have "foo-1".
func (l *List) SetHTMLCSSClass(cssClass string) {
	l.htmlCSSClass = cssClass
}

// SetOutputMirror sets an io.Writer for all the Render functions to "Write" to
// in addition to returning a string.
func (l *List) SetOutputMirror(mirror io.Writer) {
	l.outputMirror = mirror
}

// SetStyle overrides the DefaultStyle with the provided one.
func (l *List) SetStyle(style Style) {
	l.style = &style
}

// Style returns the current style.
func (l *List) Style() *Style {
	if l.style == nil {
		tempStyle := StyleDefault
		l.style = &tempStyle
	}
	return l.style
}

func (l *List) analyzeAndStringify(item interface{}) *listItem {
	itemStr := fmt.Sprint(item)
	if strings.Contains(itemStr, "\t") {
		itemStr = strings.Replace(itemStr, "\t", "    ", -1)
	}
	if strings.Contains(itemStr, "\r") {
		itemStr = strings.Replace(itemStr, "\r", "", -1)
	}
	return &listItem{
		Level: l.level,
		Text:  itemStr,
	}
}

// UnIndent un-indents the following items to appear left-shifted.
func (l *List) UnIndent() {
	if l.level > 0 {
		l.level--
	}
}

func (l *List) initForRender() {
	// pick a default style
	l.Style()

	// calculate the approximate size needed by looking at all entries
	l.approxSize = 0
	for _, item := range l.items {
		// account for the following when incrementing approxSize:
		// 1. prefix, 2. padding, 3. bullet, 4. text, 5. newline
		l.approxSize += utf8.RuneCountInString(l.style.LinePrefix)
		if item.Level > 0 {
			l.approxSize += utf8.RuneCountInString(l.style.CharItemVertical) * item.Level
		}
		l.approxSize += utf8.RuneCountInString(l.style.CharItemVertical)
		l.approxSize += utf8.RuneCountInString(item.Text)
		l.approxSize += utf8.RuneCountInString(l.style.CharNewline)
	}

	// default to a HTML CSS Class if none-defined
	if l.htmlCSSClass == "" {
		l.htmlCSSClass = DefaultHTMLCSSClass
	}
}

func (l *List) hasMoreItemsInLevel(levelIdx int, fromItemIdx int) bool {
	for idx := fromItemIdx + 1; idx >= 0 && idx < len(l.items); idx++ {
		if l.items[idx].Level < levelIdx {
			return false
		} else if l.items[idx].Level == levelIdx {
			return true
		}
	}
	return false
}

func (l *List) render(out *strings.Builder) string {
	outStr := out.String()
	if l.outputMirror != nil && len(outStr) > 0 {
		l.outputMirror.Write([]byte(outStr))
		l.outputMirror.Write([]byte("\n"))
	}
	return outStr
}

// renderHint has hints for the Render*() logic
type renderHint struct {
	isTopItem    bool
	isFirstItem  bool
	isOnlyItem   bool
	isLastItem   bool
	isBottomItem bool
}
