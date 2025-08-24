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
	"os"
)

func Formatf(msg string, a ...interface{}) string {
	return Stdout.Stdout.Formatf(msg, a...)
}

func Format(msg string) string {
	return Stdout.Stdout.Format(msg)
}

// Print formats using the default formats for its operands and writes to standard output.
// Spaces are added between operands when neither is a string.
// It returns the number of bytes written and any write error encountered.
func Print(a ...interface{}) (n int, err error) {
	return fmt.Fprint(Stdout, a...)
}

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
func Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(Stdout, format, a...)
}

// Println formats using the default formats for its operands and writes to standard output.
// Spaces are always added between operands and a newline is appended.
// It returns the number of bytes written and any write error encountered.
func Println(a ...interface{}) (n int, err error) {
	return fmt.Fprintln(Stdout, a...)
}

// Printfln formats according to a format specifier and writes to standard error output.
// A newline is appended.
// It returns the number of bytes written and any write error encountered.
func Printfln(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(Stdout, format+"\n", a...)
}

// Eprint formats using the default formats for its operands and writes to standard error output.
// Spaces are added between operands when neither is a string.
// It returns the number of bytes written and any write error encountered.
func Eprint(a ...interface{}) (n int, err error) {
	return fmt.Fprint(Stderr, a...)
}

// Eprintf formats according to a format specifier and writes to standard error output.
// It returns the number of bytes written and any write error encountered.
func Eprintf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(Stderr, format, a...)
}

// Eprintln formats using the default formats for its operands and writes to standard error output.
// Spaces are always added between operands and a newline is appended.
// It returns the number of bytes written and any write error encountered.
func Eprintln(a ...interface{}) (n int, err error) {
	return fmt.Fprintln(Stderr, a...)
}

// Eprintfln formats according to a format specifier and writes to standard error output.
// A newline is appended.
// It returns the number of bytes written and any write error encountered.
func Eprintfln(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(Stderr, format+"\n", a...)
}

var (
	DefaultStdout   *ConsoleOutput
	Stdout          *ConsoleOutput
	Stderr          *Output
	DiscardedOutput *Output
)

type Output struct {
	writer    io.Writer
	formatter *Formatter
}

func NewOutput(w io.Writer, formatter *Formatter) *Output {
	return &Output{
		writer:    w,
		formatter: formatter,
	}
}

func (o Output) IsQuiet() bool {
	return o.writer == io.Discard
}

func (o Output) Fd() uintptr {
	if fd, ok := o.writer.(FdHolder); ok {
		return fd.Fd()
	}

	return 0
}

func (o Output) Formatf(message string, a ...interface{}) string {
	bytes, err := o.formatter.FormatBytes([]byte(fmt.Sprintf(message, a...)))
	if err != nil {
		return message
	}
	return string(bytes)
}

func (o Output) Format(message string) string {
	bytes, err := o.formatter.FormatBytes([]byte(message))
	if err != nil {
		return message
	}
	return string(bytes)
}

func (o Output) Write(message []byte) (int, error) {
	if n, err := o.formatter.Format(message, o.writer); err != nil {
		return n, err
	}

	return len(message), nil
}

func (o Output) Close() error {
	if wc, ok := o.writer.(io.WriteCloser); ok {
		return wc.Close()
	}

	return nil
}

type ConsoleOutput struct {
	Stdout *Output
	Stderr *Output
}

func (cs ConsoleOutput) Fd() uintptr {
	if fd := cs.Stdout.Fd(); fd != 0 {
		return fd
	}
	if fd := cs.Stderr.Fd(); fd != 0 {
		return fd
	}

	return 0
}

func NewBufferedConsoleOutput(stdout, stderr io.Writer) *ConsoleOutput {
	output := &ConsoleOutput{
		Stdout: &Output{writer: stdout},
		Stderr: &Output{writer: stderr},
	}
	f := NewFormatter()
	f.Decorated = output.hasColorSupport()
	f.SupportsAdvancedDecoration = HasPosixColorSupport()
	output.SetFormatter(f)

	return output
}

func (cs ConsoleOutput) Write(p []byte) (int, error) {
	return cs.Stdout.Write(p)
}

func (cs ConsoleOutput) SetFormatter(formatter *Formatter) {
	cs.Stdout.formatter = formatter
	cs.Stderr.formatter = formatter
	setupLogLevelStyle(formatter)
}

func (cs ConsoleOutput) GetFormatter() *Formatter {
	return cs.Stdout.formatter
}

func (cs ConsoleOutput) SetDecorated(decorated bool) {
	cs.Stdout.formatter.Decorated = decorated
	cs.Stderr.formatter.Decorated = decorated
}

func (cs ConsoleOutput) hasColorSupport() bool {
	// Follow https://no-color.org/
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	if HasPosixColorSupport() {
		return true
	}

	return HasNativeColorSupport(cs.Stdout)
}

func setupLogLevelStyle(formatter *Formatter) {
	style := NewFormatterStyle("white", "", nil)
	formatter.SetStyle("logger_TRACE", style)
	formatter.SetStyle("logger_DEBUG", style)
	formatter.SetStyle("logger_INFO", NewFormatterStyle("green", "", nil))
	formatter.SetStyle("logger_NOTICE", NewFormatterStyle("blue", "", nil))
	formatter.SetStyle("logger_WARNING", NewFormatterStyle("cyan", "", nil))
	formatter.SetStyle("logger_ERROR", NewFormatterStyle("yellow", "", nil))
	style = NewFormatterStyle("red", "", nil)
	formatter.SetStyle("logger_CRITICAL", style)
	formatter.SetStyle("logger_ALERT", style)
	formatter.SetStyle("logger_EMERGENCY", NewFormatterStyle("white", "red", nil))
	formatter.SetStyle("header", NewFormatterStyle("cyan", "", nil))
}

func init() {
	DefaultStdout = RemapOutput(defaultOutputs())
	DiscardedOutput = &Output{
		io.Discard,
		NewFormatter(),
	}
}

func RemapOutput(out, err io.Writer) *ConsoleOutput {
	Stdout = NewBufferedConsoleOutput(out, err)
	Stderr = Stdout.Stderr
	setupLogLevelStyle(Stdout.GetFormatter())
	Logger = Logger.Output(Stderr)

	return Stdout
}
