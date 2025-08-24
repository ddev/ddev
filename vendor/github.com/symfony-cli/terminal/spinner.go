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
	"io"
	"runtime"
	"sync"
	"time"
)

// Spinner struct
type Spinner struct {
	Writer          io.Writer
	PrefixIndicator string
	PrefixText      string
	SuffixIndicator string
	SuffixText      string

	chars    []string
	delay    time.Duration
	lock     *sync.RWMutex
	active   bool
	stopChan chan struct{}
	cursor   Cursor
}

// NewSpinner creates a spinner
func NewSpinner(w io.Writer) *Spinner {
	chars := []string{"◐", "◓", "◑", "◒"}
	if runtime.GOOS == "windows" {
		chars = []string{"|", "/", "-", "\\"}
	}

	return &Spinner{
		Writer:          w,
		SuffixIndicator: "</>",
		PrefixIndicator: " <fg=yellow>",
		PrefixText:      "",
		SuffixText:      "",
		chars:           chars,
		delay:           150 * time.Millisecond,
		lock:            &sync.RWMutex{},
		active:          false,
		stopChan:        make(chan struct{}),
		cursor:          NewCursor(w),
	}
}

// Active returns whether the spinner is currently spinning
func (s *Spinner) Active() bool {
	return s.active
}

// Start starts the spinner
func (s *Spinner) Start() {
	if !Stdin.IsInteractive() {
		return
	}

	s.lock.Lock()
	if s.active {
		s.lock.Unlock()
		return
	}
	s.active = true
	s.lock.Unlock()

	go func() {
		b := bytes.Buffer{}
		cursor := Cursor{Writer: &b}

		for {
			for i := 0; i < len(s.chars); i++ {
				select {
				case <-s.stopChan:
					return
				default:
					s.lock.Lock()
					b.Reset()
					cursor.SavePosition()
					fmt.Fprintf(&b, "%s%s%s  %s%s", s.PrefixText, s.PrefixIndicator, s.chars[i], s.SuffixIndicator, s.SuffixText)
					cursor.ClearLineAfter()
					cursor.RestorePosition()
					b.WriteTo(s.Writer)
					s.lock.Unlock()
					time.Sleep(s.delay)
				}
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.active {
		s.active = false
		s.stopChan <- struct{}{}
		s.cursor.ClearLineAfter()
	}
}
