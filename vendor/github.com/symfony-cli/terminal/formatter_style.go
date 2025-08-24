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
	"os"
	"sync"
)

var (
	hrefSupport     bool
	hrefSupportOnce sync.Once
)

type FormatterStyle struct {
	color *Color
	href  string
}

func NewFormatterStyle(foreground, background string, options []string) *FormatterStyle {
	color, err := NewColor(foreground, background, options)
	if err != nil {
		panic(err)
	}
	return &FormatterStyle{color: color}
}

func (style *FormatterStyle) GetHref() string {
	return style.href
}

func (style *FormatterStyle) SetHref(href string) {
	style.href = href
}

func (style *FormatterStyle) apply(msg []byte) []byte {
	buf := bytes.NewBuffer([]byte{})
	if hasHrefSupport() && len(style.href) > 0 {
		buf.WriteString("\033]8;;")
		buf.WriteString(style.href)
		buf.WriteString("\033\\")
	}
	buf.Write(style.color.Set())
	buf.Write(msg)
	return buf.Bytes()
}

func (style *FormatterStyle) unapply() []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.Write(style.color.Unset())
	if hasHrefSupport() && len(style.href) > 0 {
		buf.WriteString("\033]8;;\033\\")
	}
	return buf.Bytes()
}

func hasHrefSupport() bool {
	hrefSupportOnce.Do(func() {
		hrefSupport = os.Getenv("TERMINAL_EMULATOR") != "JetBrains-JediTerm" && os.Getenv("KONSOLE_VERSION") == ""
	})
	return hrefSupport
}
