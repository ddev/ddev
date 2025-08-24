/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package terminal

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"sync"
)

type styleStack []*FormatterStyle

func (s *styleStack) push(v *FormatterStyle) {
	*s = append(*s, v)
}

func (s *styleStack) pop(style *FormatterStyle) *FormatterStyle {
	n := len(*s)

	if n == 0 {
		return NewFormatterStyle("", "", nil)
	}

	if style == nil {
		res := (*s)[n-1]
		*s = (*s)[:n-1]

		return res
	}

	for index := len(*s) - 1; index >= 0; index-- {
		stackedStyle := (*s)[index]
		if bytes.Equal(stackedStyle.apply([]byte(``)), style.apply([]byte(``))) {
			*s = (*s)[:index]

			return stackedStyle
		}
	}

	panic("Incorrectly nested style tag found.")
}

func (s *styleStack) current() *FormatterStyle {
	n := len(*s)

	if n == 0 {
		return NewFormatterStyle("", "", nil)
	}

	return (*s)[n-1]
}

const tagRegex = "[a-z][^<>]*"

var (
	FormattingRegexp = regexp.MustCompile("(?i)<((" + tagRegex + ")?|/(" + tagRegex + ")?)>")
	StyleRegexp      = regexp.MustCompile("(?i)([^=]+)=([^;]+)(;|$)")
	EscapingRegexp   = regexp.MustCompile("([^\\\\]?)<")
)

func Escape(msg []byte) []byte {
	msg = EscapingRegexp.ReplaceAll(msg, []byte("$1\\<"))

	return EscapeTrailingBackslash(msg)
}

func EscapeTrailingBackslash(msg []byte) []byte {
	l := len(msg)
	if msg[l-1] == '\\' {
		msg = bytes.TrimRight(msg, "\\")
		msg = append(msg, bytes.Repeat([]byte("<"), 2*(l-len(msg)))...)
	}

	return msg
}

type Formatter struct {
	Decorated                  bool
	SupportsAdvancedDecoration bool

	stack  styleStack
	lock   sync.RWMutex
	styles map[string]*FormatterStyle
}

func NewFormatter() *Formatter {
	formatter := &Formatter{
		Decorated:                  true,
		SupportsAdvancedDecoration: true,
		styles:                     make(map[string]*FormatterStyle),
	}
	formatter.SetStyle("success", NewFormatterStyle("white", "green", nil))
	formatter.SetStyle("error", NewFormatterStyle("white", "red", nil))
	formatter.SetStyle("info", NewFormatterStyle("green", "", nil))
	formatter.SetStyle("comment", NewFormatterStyle("yellow", "", nil))
	formatter.SetStyle("question", NewFormatterStyle("black", "cyan", nil))
	formatter.SetStyle("warning", NewFormatterStyle("black", "yellow", nil))

	return formatter
}

func (formatter *Formatter) SetStyle(name string, style *FormatterStyle) {
	formatter.lock.Lock()
	defer formatter.lock.Unlock()

	formatter.styles[strings.ToLower(name)] = style
}

func (formatter *Formatter) AddAlias(from, to string) {
	formatter.lock.Lock()
	defer formatter.lock.Unlock()

	formatter.styles[strings.ToLower(from)] = formatter.styles[strings.ToLower(to)]
}

func (formatter *Formatter) HasStyle(name string) bool {
	formatter.lock.RLock()
	defer formatter.lock.RUnlock()

	_, hasStyle := formatter.styles[strings.ToLower(name)]

	return hasStyle
}

func (formatter *Formatter) FormatBytes(msg []byte) ([]byte, error) {
	var buf []byte
	buffer := bytes.NewBuffer(buf)
	_, err := formatter.Format(msg, buffer)

	return buffer.Bytes(), err
}

func (formatter *Formatter) Format(msg []byte, w io.Writer) (written int, err error) {
	formatter.lock.Lock()
	defer formatter.lock.Unlock()

	offset := 0
	output := new(bytes.Buffer)
	// msg length is a good guess to preallocate buffer
	output.Grow(len(msg))

	matches := FormattingRegexp.FindAllIndex(msg, -1)

	for _, match := range matches {
		spec := msg[match[0]:match[1]]

		if match[0] != 0 && msg[match[0]-1] == '\\' {
			continue
		}

		if match[0] > offset {
			_, err = output.Write(formatter.applyCurrentStyle(msg[offset:match[0]]))
			written += match[0] - offset
			if err != nil {
				return
			}
		}

		offset = match[1]

		open := spec[1] != '/'
		tag := ""
		if open {
			tag = string(spec[1 : len(spec)-1])
		} else {
			tag = string(spec[2 : len(spec)-1])
		}

		style, err1 := formatter.createStyleFromString(tag)
		if err1 != nil {
			return written, err1
		}

		if !open {
			if style != nil || len(tag) == 0 {
				style = formatter.stack.pop(style)
			}
			if style != nil {
				if formatter.Decorated {
					output.Write(style.unapply())
				}
				continue
			}
		} else {
			if style != nil {
				formatter.stack.push(style)
				continue
			}

			spec = formatter.applyCurrentStyle(spec)
		}

		_, err = output.Write(spec)
		written += len(spec)
		if err != nil {
			return
		}
	}

	if offset < len(msg) {
		// remaining bits
		_, err = output.Write(formatter.applyCurrentStyle(msg[offset:]))
		written += len(msg) - offset
		if err != nil {
			return
		}
	}

	if _, err = output.Write(formatter.unapplyCurrentStyle()); err != nil {
		return
	}

	msg = output.Bytes()
	if bytes.Contains(msg, []byte("<<")) {
		msg = bytes.Replace(msg, []byte("\\<"), []byte("\\<"), -1)
		msg = bytes.Replace(msg, []byte("<<"), []byte("\\"), -1)
	}

	msg = bytes.Replace(msg, []byte("\\<"), []byte("<"), -1)
	_, err = w.Write(msg)

	return
}

func (formatter *Formatter) applyCurrentStyle(msg []byte) []byte {
	if formatter.Decorated {
		return formatter.stack.current().apply(msg)
	}

	return msg
}

func (formatter *Formatter) unapplyCurrentStyle() []byte {
	if formatter.Decorated {
		return formatter.stack.current().unapply()
	}

	return []byte{}
}

func (formatter *Formatter) createStyleFromString(format string) (*FormatterStyle, error) {
	if style, ok := formatter.styles[strings.ToLower(format)]; ok {
		return style, nil
	}

	matches := StyleRegexp.FindAllStringSubmatch(format, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	foreground := ""
	background := ""
	options := []string{}
	href := ""
	for _, match := range matches {
		switch strings.ToLower(match[1]) {
		case "href":
			if formatter.SupportsAdvancedDecoration {
				href = match[2]
			}
		case "fg":
			foreground = match[2]
		case "bg":
			background = match[2]
		case "options":
			options = strings.Split(match[2], ",")
		default:
			return nil, nil
		}
	}

	color, err := NewColor(foreground, background, options)
	if err != nil {
		return nil, err
	}
	style := &FormatterStyle{color: color}
	style.SetHref(href)

	return style, nil
}
