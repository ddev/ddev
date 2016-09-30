package utils

import (
	"errors"
	"os"
	"os/exec"
)

// GetHomeDir uses the $HOME var to return the user's home directory
func GetHomeDir() (string, error) {
	homedir := os.Getenv("HOME")
	if homedir == "" {
		return "", errors.New("Standard methods to locate user's home firectory failed. Please set $HOME and try again.")
	}

	return homedir, nil
}

// RunCommand runs a command on the host system.
func RunCommand(command string, args []string) (string, error) {
	out, err := exec.Command(
		command,
		args...,
	).CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

// FileExists checks a file's existence
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
