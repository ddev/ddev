package util

import (
	"math/rand"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"log"
	"path"
)

// Failed will print an red error message and exit with failure.
func Failed(format string, a ...interface{}) {
	color.Red(format, a...)
	os.Exit(1)
}

// Warning will present the user with warning text.
func Warning(format string, a ...interface{}) {
	color.Yellow(format, a...)
}

// Success will indicate an operation succeeded with colored confirmation text.
func Success(format string, a ...interface{}) {
	color.Cyan(format, a...)
}

// FormatPlural is a simple wrapper which returns different strings based on the count value.
func FormatPlural(count int, single string, plural string) string {
	if count == 1 {
		return single
	}
	return plural
}

var letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// setLetterBytes exists solely so that tests can override the default characters used by
// RandString. It should probably be avoided for 'normal' operations.
func setLetterBytes(lb string) {
	letterBytes = lb
}

// RandString returns a random string of given length n.
func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// GetGlobalDdevDir returns ~/.ddev, the global caching directory
func GetGlobalDdevDir() string {
	userHome, err := homedir.Dir()
	if err != nil {
		log.Fatal("could not get home directory for current user. is it set?")
	}
	ddevDir := path.Join(userHome, ".ddev")
	return ddevDir
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
