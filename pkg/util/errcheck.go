package util

import (
	"io"

	log "github.com/sirupsen/logrus"
)

// CheckErr exits with a log.Fatal() if an error is encountered.
// It is normally used for errors that we never expect to happen,
// and don't have any normal handling technique.
// From https://davidnix.io/post/error-handling-in-go/
func CheckErr(err error) {
	if err != nil {
		log.Panic("CheckErr(): ERROR:", err)
	}
}

// CheckClose is used to check the return from Close in a defer statement.
// From https://groups.google.com/d/msg/golang-nuts/-eo7navkp10/BY3ym_vMhRcJ
func CheckClose(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Println("Failed to close deferred io.Closer, err: ", err)
	}
}
