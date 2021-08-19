package list

import (
	"html"
	"strconv"
	"strings"
)

// RenderHTML renders the List in the HTML format. Example:
//  <ul class="go-pretty-table">
//    <li>Game Of Thrones</li>
//    <ul class="go-pretty-table-1">
//      <li>Winter</li>
//      <li>Is</li>
//      <li>Coming</li>
//      <ul class="go-pretty-table-2">
//        <li>This</li>
//        <li>Is</li>
//        <li>Known</li>
//      </ul>
//    </ul>
//    <li>The Dark Tower</li>
//    <ul class="go-pretty-table-1">
//      <li>The Gunslinger</li>
//    </ul>
//  </ul>
func (l *List) RenderHTML() string {
	l.initForRender()

	var out strings.Builder
	if len(l.items) > 0 {
		l.htmlRenderRecursively(&out, 0, l.items[0])
	}
	return l.render(&out)
}

func (l *List) htmlRenderRecursively(out *strings.Builder, idx int, item *listItem) int {
	linePrefix := strings.Repeat("  ", item.Level)

	out.WriteString(linePrefix)
	out.WriteString("<ul class=\"")
	out.WriteString(l.htmlCSSClass)
	if item.Level > 0 {
		out.WriteRune('-')
		out.WriteString(strconv.Itoa(item.Level))
	}
	out.WriteString("\">\n")
	var numItemsRendered int
	for itemIdx := idx; itemIdx < len(l.items); itemIdx++ {
		if l.items[itemIdx].Level == item.Level {
			out.WriteString(linePrefix)
			out.WriteString("  <li>")
			out.WriteString(strings.Replace(html.EscapeString(l.items[itemIdx].Text), "\n", "<br/>", -1))
			out.WriteString("</li>\n")
			numItemsRendered++
		} else if l.items[itemIdx].Level > item.Level { // indent
			numItemsRenderedRecursively := l.htmlRenderRecursively(out, itemIdx, l.items[itemIdx])
			numItemsRendered += numItemsRenderedRecursively
			itemIdx += numItemsRenderedRecursively - 1
			if numItemsRendered > 0 {
				out.WriteRune('\n')
			}
		} else { // un-indent
			break
		}
	}
	out.WriteString(linePrefix)
	out.WriteString("</ul>")
	return numItemsRendered
}
