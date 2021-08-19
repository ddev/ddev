package list

import (
	"strings"
	"unicode/utf8"
)

// Render renders the List in a human-readable "pretty" format. Example:
//  * Game Of Thrones
//    * Winter
//    * Is
//    * Coming
//      * This
//      * Is
//      * Known
//  * The Dark Tower
//    * The Gunslinger
func (l *List) Render() string {
	l.initForRender()

	var out strings.Builder
	out.Grow(l.approxSize)
	for idx, item := range l.items {
		hint := renderHint{
			isTopItem:    bool(idx == 0),
			isFirstItem:  bool(idx == 0 || item.Level > l.items[idx-1].Level),
			isLastItem:   !l.hasMoreItemsInLevel(item.Level, idx),
			isBottomItem: bool(idx == len(l.items)-1),
		}
		if hint.isFirstItem && hint.isLastItem {
			hint.isOnlyItem = true
		}
		l.renderItem(&out, idx, item, hint)
	}
	return l.render(&out)
}

func (l *List) renderItem(out *strings.Builder, idx int, item *listItem, hint renderHint) {
	// when working on item number 2 or more, render a newline first
	if idx > 0 {
		out.WriteRune('\n')
	}

	// format item.Text as directed in l.style
	itemStr := l.style.Format.Apply(item.Text)

	// convert newlines if newlines are not "\n" in l.style
	if strings.Contains(itemStr, "\n") && l.style.CharNewline != "\n" {
		itemStr = strings.Replace(itemStr, "\n", l.style.CharNewline, -1)
	}

	// render the item.Text line by line
	for lineIdx, lineStr := range strings.Split(itemStr, "\n") {
		if lineIdx > 0 {
			out.WriteRune('\n')
		}

		// render the prefix or the leading text before the actual item
		l.renderItemBulletPrefix(out, idx, item.Level, lineIdx, hint)
		l.renderItemBullet(out, idx, item.Level, lineIdx, hint)

		// render the actual item
		out.WriteString(lineStr)
	}
}

func (l *List) renderItemBullet(out *strings.Builder, itemIdx int, itemLevel int, lineIdx int, hint renderHint) {
	if lineIdx > 0 {
		// multi-line item.Text
		if hint.isLastItem {
			out.WriteString(strings.Repeat(" ", utf8.RuneCountInString(l.style.CharItemVertical)))
		} else {
			out.WriteString(l.style.CharItemVertical)
		}
	} else {
		// single-line item.Text (or first line of a multi-line item.Text)
		if hint.isOnlyItem {
			if hint.isTopItem {
				out.WriteString(l.style.CharItemSingle)
			} else {
				out.WriteString(l.style.CharItemBottom)
			}
		} else if hint.isTopItem {
			out.WriteString(l.style.CharItemTop)
		} else if hint.isFirstItem {
			out.WriteString(l.style.CharItemFirst)
		} else if hint.isBottomItem || hint.isLastItem {
			out.WriteString(l.style.CharItemBottom)
		} else {
			out.WriteString(l.style.CharItemMiddle)
		}
		out.WriteRune(' ')
	}
}

func (l *List) renderItemBulletPrefix(out *strings.Builder, itemIdx int, itemLevel int, lineIdx int, hint renderHint) {
	// write a prefix if one has been set in l.style
	if l.style.LinePrefix != "" {
		out.WriteString(l.style.LinePrefix)
	}

	// render spaces and connectors until the item's position
	for levelIdx := 0; levelIdx < itemLevel; levelIdx++ {
		if l.hasMoreItemsInLevel(levelIdx, itemIdx) {
			out.WriteString(l.style.CharItemVertical)
		} else {
			out.WriteString(strings.Repeat(" ", utf8.RuneCountInString(l.style.CharItemVertical)))
		}
	}
}
