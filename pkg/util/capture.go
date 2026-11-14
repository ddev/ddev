package util

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/ddev/ddev/pkg/output"
)

// capturePipe returns a pipe's write end and a function that closes it and
// returns everything written. Reading starts at once, so output larger than
// the pipe buffer can't block the writer.
func capturePipe() (*os.File, func() string) {
	r, w, _ := os.Pipe()
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		CheckErr(err)
		outC <- buf.String()
	}()
	return w, func() string {
		CheckClose(w)
		return <-outC
	}
}

// CaptureUserOut captures output written to UserOut to a string.
// Capturing starts when it is called. It returns an anonymous function that
// when called, will return a string containing the output during capture, and
// revert once again to the original value of UserOut.Out.
func CaptureUserOut() func() string {
	old := output.UserOut.Out // keep backup of the real stdout
	w, finish := capturePipe()
	output.UserOut.Out = w

	return func() string {
		output.UserOut.Out = old // restoring the real stdout
		return finish()
	}
}

// CaptureUserErr captures output written to UserErr to a string.
// Capturing starts when it is called. It returns an anonymous function that
// when called, will return a string containing the output during capture, and
// revert once again to the original value of UserErr.Out.
func CaptureUserErr() func() string {
	old := output.UserErr.Out // keep backup of the real stderr
	w, finish := capturePipe()
	output.UserErr.Out = w

	return func() string {
		output.UserErr.Out = old // restoring the real stderr
		return finish()
	}
}

// CaptureStdOut captures Stdout to a string. Capturing starts when it is called. It returns an anonymous function that when called, will return a string
// containing the output during capture, and revert once again to the original value of os.StdOut.
func CaptureStdOut() func() string {
	old := os.Stdout // keep backup of the real stdout
	w, finish := capturePipe()
	os.Stdout = w

	return func() string {
		os.Stdout = old // restoring the real stdout
		return finish()
	}
}

// CaptureOutputToFile captures Stdout to a string. Capturing starts when it is called. It returns an anonymous function that when called, will return a string
// containing the output during capture, and revert once again to the original value of os.StdOut.
func CaptureOutputToFile() (func() string, error) {
	oldStdout := os.Stdout // keep backup of the real stdout
	oldStderr := os.Stderr
	f, err := os.CreateTemp("", "CaptureOutputToFile")
	if err != nil {
		return nil, err
	}
	os.Stdout = f
	os.Stderr = f

	return func() string {
		_ = f.Close()
		os.Stdout = oldStdout // restoring the real stdout
		os.Stderr = oldStderr
		out, err := os.ReadFile(f.Name())
		if err != nil {
			out = fmt.Appendf(nil, "failed to read file: %v", err)
		}
		defer func() {
			_ = os.RemoveAll(f.Name())
		}()
		return string(out)
	}, nil
}
