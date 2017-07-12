package util

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"log"
	"path"

	"github.com/drud/ddev/pkg/util"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Failed will print an red error message and exit with failure.
func Failed(format string, a ...interface{}) {
	Error(format, a...)
	os.Exit(1)
}

// Error will print an red error message but will not exit.
func Error(format string, a ...interface{}) {
	color.Red(format, a...)
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

	// Create the directory if it is not already present.
	if _, err := os.Stat(ddevDir); os.IsNotExist(err) {
		err = os.MkdirAll(ddevDir, 0700)
		if err != nil {
			util.Failed("Failed to create required directory %s, err: %v", ddevDir, err)
		}
	}
	return ddevDir
}

// AskForConfirmation requests a y/n from user.
func AskForConfirmation() bool {
	response := GetInput("")
	okayResponses := []string{"y", "yes"}
	nokayResponses := []string{"n", "no", ""}
	responseLower := strings.ToLower(response)

	if containsString(okayResponses, responseLower) {
		return true
	} else if containsString(nokayResponses, responseLower) {
		return false
	} else {
		fmt.Println("Please type yes or no and then press enter:")
		return AskForConfirmation()
	}
}

// containsString returns true if slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}
