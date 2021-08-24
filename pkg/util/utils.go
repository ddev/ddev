package util

import (
	"fmt"
	"golang.org/x/text/runes"

	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"math"
	"math/rand"
	osexec "os/exec"
	"os/user"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/drud/ddev/pkg/output"
	"github.com/fatih/color"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Failed will print a red error message and exit with failure.
func Failed(format string, a ...interface{}) {
	if a != nil {
		//output.UserOut.Fatalf(format, a...)
		output.UserErr.Fatalf(format, a...)
		//output.UserOut.WithField("level", "fatal").Fatalf(format, a...)
	} else {
		output.UserErr.Fatal(format)
		//output.UserOut.WithField("level", "fatal").Fatal(format)
	}
}

// Error will print an red error message but will not exit.
func Error(format string, a ...interface{}) {
	if a != nil {
		output.UserErr.Errorf(format, a...)
	} else {
		output.UserErr.Error(format)
	}
}

// Warning will present the user with warning text.
func Warning(format string, a ...interface{}) {
	if a != nil {
		output.UserErr.Warnf(format, a...)
	} else {
		output.UserErr.Warn(format)
	}
}

// Success will indicate an operation succeeded with colored confirmation text.
func Success(format string, a ...interface{}) {
	format = color.CyanString(format)
	if a != nil {
		output.UserOut.Infof(format, a...)
	} else {
		output.UserOut.Info(format)
	}
}

// Output controlled by DDEV_DEBUG environment variable
func Debug(format string, a ...interface{}) {
	if globalconfig.DdevDebug {
		output.UserOut.Debugf(format, a...)
	}
}

// FormatPlural is a simple wrapper which returns different strings based on the count value.
func FormatPlural(count int, single string, plural string) string {
	if count == 1 {
		return single
	}
	return plural
}

var letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// SetLetterBytes exists solely so that tests can override the default characters used by
// RandString. It should probably be avoided for 'normal' operations.
// this is actually used in utils_test.go (test only) so we set nolint on it.
// nolint: deadcode
func SetLetterBytes(lb string) {
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

// AskForConfirmation requests a y/n from user.
func AskForConfirmation() bool {
	response := GetInput("")
	okayResponses := []string{"y", "yes"}
	nokayResponses := []string{"n", "no", ""}
	responseLower := strings.ToLower(response)

	if nodeps.ArrayContainsString(okayResponses, responseLower) {
		return true
	} else if nodeps.ArrayContainsString(nokayResponses, responseLower) {
		return false
	} else {
		output.UserOut.Println("Please type yes or no and then press enter:")
		return AskForConfirmation()
	}
}

// MapKeysToArray takes the keys of the map and turns them into a string array
func MapKeysToArray(mapWithKeys map[string]interface{}) []string {
	result := make([]string, 0, len(mapWithKeys))
	for v := range mapWithKeys {
		result = append(result, v)
	}
	return result
}

// GetContainerUIDGid returns the uid and gid (and string forms) to be used running most containers
func GetContainerUIDGid() (uidStr string, gidStr string, username string) {
	curUser, err := user.Current()
	if err != nil {
		Failed("Unable to determine username and related UID, etc. Please at least set $USER environment variable: %v", err)
	}
	uidStr = curUser.Uid
	gidStr = curUser.Gid
	username = curUser.Username
	// Remove at least spaces that aren't allowed in linux usernames and can appear in windows
	// Example problem usernames from https://stackoverflow.com/questions/64933879/docker-ddev-unicodedecodeerror-utf-8-codec-cant-decode-byte-0xe9-in-positio/64934264#64934264
	// 'André Kraus', 'Mück'

	// Normalize username per https://stackoverflow.com/a/65981868/215713
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	username, _, _ = transform.String(t, username)

	username = strings.Replace(username, " ", "", -1)
	username = strings.ToLower(username)

	// If we have a numeric username it's going to create havoc, so
	// change it into "a" + number
	// Example in https://github.com/drud/ddev/issues/3187 - username="310822", uid=1663749668, gid=1240132652
	if !nodeps.IsLetter(string(username[0])) {
		username = "a" + username
	}
	//// Windows userids are non-numeric,
	//// so we have to run as arbitrary user 1000. We may have a host uidStr/gidStr greater in other contexts,
	//// 1000 seems not to cause file permissions issues at least on docker-for-windows.
	if runtime.GOOS == "windows" {
		uidStr = "1000"
		gidStr = "1000"
		parts := strings.Split(curUser.Username, `\`)
		username = parts[len(parts)-1]
		username = strings.ReplaceAll(username, " ", "")
		username = strings.ToLower(username)
	}
	return uidStr, gidStr, username

}

// IsCommandAvailable uses shell's "command" to find out if a command is available
// https://siongui.github.io/2018/03/16/go-check-if-command-exists/
// This lives here instead of in fileutil to avoid unecessary import cycles.
func IsCommandAvailable(cmdName string) bool {
	_, err := osexec.LookPath(cmdName)
	if err == nil {
		return true
	}
	return false
}

// GetFirstWord just returns the first space-separated word in a string.
func GetFirstWord(s string) string {
	arr := strings.Split(s, " ")
	return arr[0]
}

// FindBashPath returns the PATH to bash on any system
// on Windows preferring git-bash
// On Windows we'll need the path to bash to execute anything.
// Returns empty string if not found, path if found
func FindBashPath() string {
	if runtime.GOOS != "windows" {
		return "bash"
	}
	windowsBashPath, err := osexec.LookPath(`C:\Program Files\Git\bin\bash.exe`)
	if err != nil {
		// This one could come back with the WSL bash, in which case we may have some trouble.
		windowsBashPath, err = osexec.LookPath("bash.exe")
		if err != nil {
			fmt.Println("Not loading custom commands; bash is not in PATH")
			return ""
		}
	}
	return windowsBashPath
}

// TimeTrack determines the amount of time a function takes to return. Timing starts when it is called.
// It returns an anonymous function that, when called, will print the elapsed run time.
// It tracks if DDEV_VERBOSE is set
func TimeTrack(start time.Time, name string) func() {
	if globalconfig.DdevVerbose {
		logrus.Printf("starting %s at %v\n", name, start.Format("15:04:05.000000000"))
		return func() {
			if globalconfig.DdevVerbose {
				elapsed := time.Since(start)
				logrus.Printf("PERF: %s took %.2fs", name, elapsed.Seconds())
			}
		}
	}
	return func() {
	}
}

// ElapsedTime is an easy way to report how long something took.
// It returns an anonymous function that, when called, will return the elapsed seconds.
func ElapsedTime(start time.Time) func() float64 {
	return func() float64 {
		elapsed := time.Since(start)
		return elapsed.Seconds()
	}
}

// ElapsedDuration is an easy way to report how long something took.
// It returns an anonymous function that, when called, will return the elapsed duration.
func ElapsedDuration(start time.Time) func() time.Duration {
	return func() time.Duration {
		return time.Since(start)
	}
}

// FormatDuration formats with 5m20s instead of lots of decimal points
// Based on https://stackoverflow.com/a/47342272/215713
func FormatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(math.Round(d.Seconds())) - (minutes * 60)
	if minutes == 0 {
		return fmt.Sprintf("%ds", seconds)
	}
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// IsBeforeCutoffTime returns true if the current time is before
// the cutoff time, in format "01 Jan 21 10:00 UTC"
func IsBeforeCutoffTime(cutoff string) bool {
	cutoffTime, err := time.Parse(time.RFC822, cutoff)
	if err != nil {
		output.UserErr.Printf("Failed to parse cutoffTime %s: %v", cutoffTime, err)
	}
	currentTime := time.Now()
	if currentTime.Before(cutoffTime) {
		return true
	}
	return false
}
