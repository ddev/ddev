package util

import (
	"github.com/drud/ddev/pkg/nodeps"
	"math/rand"
	"os"
	osexec "os/exec"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/drud/ddev/pkg/output"
	"github.com/fatih/color"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Failed will print a red error message and exit with failure.
func Failed(format string, a ...interface{}) {
	if a != nil {
		output.UserOut.Fatalf(format, a...)
	} else {
		output.UserOut.Fatal(format)
	}
}

// Error will print an red error message but will not exit.
func Error(format string, a ...interface{}) {
	if a != nil {
		output.UserOut.Errorf(format, a...)
	} else {
		output.UserOut.Error(format)
	}
}

// Warning will present the user with warning text.
func Warning(format string, a ...interface{}) {
	if a != nil {
		output.UserOut.Warnf(format, a...)
	} else {
		output.UserOut.Warn(format)
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

// GetContainerUIDGid() returns the uid and gid (and string forms) to be used running most containers.
func GetContainerUIDGid() (uid int, gid int, uidStr string, gidStr string) {
	var uidInt, gidInt int
	curUser, err := user.Current()
	CheckErr(err)

	uidStr = curUser.Uid
	gidStr = curUser.Gid
	// For windows the uidStr/gidStr are usually way outside linux range (ends at 60000)
	// so we have to run as arbitrary user 1000. We may have a host uidStr/gidStr greater in other contexts,
	// 1000 seems not to cause file permissions issues at least on docker-for-windows.
	// TODO: For large macOS UIDs we might be better to add the UID to /etc/passwd at startup
	if uidInt, err = strconv.Atoi(curUser.Uid); err != nil || uidInt > 60000 {
		uidStr = "1000"
		uidInt = 1000
	}
	if gidInt, err = strconv.Atoi(curUser.Gid); err != nil || gidInt > 60000 {
		gidStr = "1000"
		gidInt = 1000
	}
	return uidInt, gidInt, uidStr, gidStr

}

// IsDockerToolbox detects if the running docker is docker toolbox
// It shouldn't be run much as it requires actually running the executable.
// This lives here instead of in dockerutils to avoid unecessary import cycles.
// Inspired by https://stackoverflow.com/questions/43242218/how-can-a-script-distinguish-docker-toolbox-and-docker-for-windows
func IsDockerToolbox() bool {
	dockerToolboxPath := os.Getenv("DOCKER_TOOLBOX_INSTALL_PATH")
	if dockerToolboxPath != "" {
		return true
	}
	return false
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
