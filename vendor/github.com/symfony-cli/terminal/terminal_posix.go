//go:build !windows
// +build !windows

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
	"os"

	isattypkg "github.com/mattn/go-isatty"
	"golang.org/x/term"
)

var (
	IsTTY   = isattypkg.IsTerminal
	MakeRaw = term.MakeRaw
	Restore = term.Restore
)

func GetSize() (width, height int) {
	for _, f := range []*os.File{os.Stderr, os.Stdout, os.Stdin} {
		w, h, err := term.GetSize(int(f.Fd()))
		if err != nil {
			continue
		}

		if w > 0 && h > 0 {
			return w, h
		}
	}

	return defaultWidth, defaultHeight
}

func IsCygwinTTY(fd uintptr) bool {
	return false
}
