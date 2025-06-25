package util

import (
	"github.com/ddev/ddev/pkg/output"
	"io"
)

// CheckErr exits with a log.Fatal() if an error is encountered.
// It is normally used for errors that we never expect to happen,
// and don't have any normal handling technique.
// From https://davidnix.io/post/error-handling-in-go/
func CheckErr(err error) {
	if err != nil {
		output.UserErr.Panic("CheckErr(): ERROR:", err)
	}
}

// CheckClose is used to check the return from Close in a defer statement.
// From https://groups.google.com/d/msg/golang-nuts/-eo7navkp10/BY3ym_vMhRcJ
func CheckClose(c io.Closer) {
	err := c.Close()
	if err != nil {
		output.UserErr.Println("Failed to close deferred io.Closer, err: ", err)
	}
}
