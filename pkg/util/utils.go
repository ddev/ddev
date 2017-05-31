package util

import (
	"math/rand"
	"os"
	"time"

	"github.com/fatih/color"
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

// RandomString returns a random string of len strlen
func RandomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
