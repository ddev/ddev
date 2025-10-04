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
	"io"
)

const (
	ErrNotATTY                  = termErr(0)
	defaultWidth, defaultHeight = 80, 20
)

type FdHolder interface {
	Fd() uintptr
}

func IsTerminal(stream interface{}) bool {
	output, streamIsFile := stream.(FdHolder)
	return streamIsFile && IsTTY(output.Fd())
}

type FdReader interface {
	io.Reader
	FdHolder
}

type termErr int

func (e termErr) Error() string {
	switch e {
	case 0:
		return "not a TTY"
	}
	return "undefined terminal error"
}
