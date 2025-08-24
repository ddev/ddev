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
	"fmt"
	"regexp"
	"strings"
)

const maxLineLength = 120

type style struct {
	stdout *ConsoleOutput
	stdin  *Input
}

func SymfonyStyle(stdout *ConsoleOutput, stdin *Input) style {
	return style{stdout, stdin}
}

func (s style) Title(message string) {
	stripped := regexp.MustCompile(`<.+?>`).ReplaceAllString(message, "")
	stripped = regexp.MustCompile(`<.*?/>`).ReplaceAllString(stripped, "")
	fmt.Fprintf(s.stdout, "<comment>%s</>\n", message)
	fmt.Fprintf(s.stdout, "<comment>%s</>\n", strings.Repeat("=", len(stripped)))
	fmt.Fprintln(s.stdout)
}

func (s style) Section(message string) {
	stripped := regexp.MustCompile(`<.+?>`).ReplaceAllString(message, "")
	stripped = regexp.MustCompile(`<.*?/>`).ReplaceAllString(stripped, "")
	fmt.Fprintf(s.stdout, "<comment>%s</>\n", message)
	fmt.Fprintf(s.stdout, "<comment>%s</>\n", strings.Repeat("-", len(stripped)))
	fmt.Fprintln(s.stdout)
}

func (s style) Block(messages []string, typePrefix, style, prefix string, padding bool) {
	fmt.Fprintln(s.stdout, s.createBlock(messages, typePrefix, style, prefix, padding))
}

func (s style) Comment(message string) {
	s.Block([]string{message}, "", "", "<fg=default;bg=default> // </>", false)
}

func (s style) Success(message string) {
	s.Block([]string{message}, "OK", "fg=black;bg=green", " ", true)
}

func (s style) Error(message string) {
	s.Block([]string{message}, "ERROR", "fg=white;bg=red", " ", true)
}

func (s style) Warning(message string) {
	s.Block([]string{message}, "WARNING", "fg=black;bg=yellow", " ", true)
}

func (s style) Note(message string) {
	s.Block([]string{message}, "NOTE", "fg=yellow", " ! ", true)
}

func (s style) Caution(message string) {
	s.Block([]string{message}, "CAUTION", "fg=white;bg=red", " ! ", true)
}

func (s style) createBlock(messages []string, typePrefix, style, prefix string, padding bool) string {
	var buf bytes.Buffer

	width, _ := GetSize()
	if width > maxLineLength {
		width = maxLineLength
	}

	fullPadding := strings.Repeat(" ", width)

	if typePrefix != "" {
		typePrefix = fmt.Sprintf("[%s] ", typePrefix)
	}

	width -= len(prefix) + len(typePrefix)

	lines := []string{}

	for _, msg := range messages {
		l, _ := splitsBlockLines(msg, width-2)
		lines = append(lines, l...)
	}

	if padding {
		if style != "" {
			buf.WriteString(fmt.Sprintf("<%s>", style))
		}
		buf.WriteString(fullPadding)
		if style != "" {
			buf.WriteString("</>")
		}
		buf.WriteString("\n")
	}

	for i, line := range lines {
		if style != "" {
			buf.WriteString(fmt.Sprintf("<%s>", style))
		}

		buf.WriteString(prefix)

		if typePrefix != "" && i == 0 {
			buf.WriteString(typePrefix)
		}

		lenLine, _ := Stdout.GetFormatter().Format([]byte(line), &buf)
		if width-lenLine > 0 {
			buf.WriteString(strings.Repeat(" ", width-lenLine))
		}

		if typePrefix != "" && i != 0 {
			buf.WriteString(strings.Repeat(" ", len(typePrefix)))
		}

		if style != "" {
			buf.WriteString("</>")
		}
		buf.WriteString("\n")
	}

	if padding {
		if style != "" {
			buf.WriteString(fmt.Sprintf("<%s>", style))
		}
		buf.WriteString(fullPadding)
		if style != "" {
			buf.WriteString("</>")
		}
		buf.WriteString("\n")
	}

	return buf.String()
}
