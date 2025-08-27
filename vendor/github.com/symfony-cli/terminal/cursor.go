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
	"fmt"
	"io"
)

type Cursor struct {
	Writer io.Writer
}

func NewCursor(w io.Writer) Cursor {
	return Cursor{Writer: w}
}

func (c Cursor) MoveUp(lines int) Cursor {
	fmt.Fprintf(c.Writer, "\x1b[%dA", lines)
	return c
}

func (c Cursor) MoveDown(lines int) Cursor {
	fmt.Fprintf(c.Writer, "\x1b[%dB", lines)
	return c
}

func (c Cursor) MoveRight(columns int) Cursor {
	fmt.Fprintf(c.Writer, "\x1b[%dC", columns)
	return c
}

func (c Cursor) MoveLeft(columns int) Cursor {
	fmt.Fprintf(c.Writer, "\x1b[%dD", columns)
	return c
}

func (c Cursor) MoveToColumn(column int) Cursor {
	fmt.Fprintf(c.Writer, "\x1b[%dG", column)
	return c
}

func (c Cursor) MoveToPosition(column, row int) Cursor {
	fmt.Fprintf(c.Writer, "\x1b[%d;%dH", row+1, column)
	return c
}

func (c Cursor) SavePosition() Cursor {
	fmt.Fprint(c.Writer, "\x1b7\x1b[s")
	return c
}

func (c Cursor) RestorePosition() Cursor {
	fmt.Fprint(c.Writer, "\x1b[u\x1b8")
	return c
}

func (c Cursor) Hide() Cursor {
	fmt.Fprint(c.Writer, "\x1b[?25l")
	return c
}

func (c Cursor) Show() Cursor {
	fmt.Fprint(c.Writer, "\x1b[?25h\x1b[?0c")
	return c
}

// ClearLine clears all the output from the current line.
func (c Cursor) ClearLine() Cursor {
	fmt.Fprint(c.Writer, "\r\x1b[2K")
	return c
}

// ClearLine clears all the output from the current line after the current position.
func (c Cursor) ClearLineAfter() Cursor {
	fmt.Fprint(c.Writer, "\x1b[K")
	return c
}

// ClearOutput clears all the output from the cursors' current position to the end of the screen.
func (c Cursor) ClearOutput() Cursor {
	fmt.Fprint(c.Writer, "\x1b[0J")
	return c
}

// ClearOutput clears the entire screen.
func (c Cursor) ClearScreen() Cursor {
	fmt.Fprint(c.Writer, "\x1b[0J")
	return c
}
