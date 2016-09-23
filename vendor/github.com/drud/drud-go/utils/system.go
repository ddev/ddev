package utils

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
)

// GetHomeDir uses the os/user module to locate user's home directory. If this failes check the contents of $HOME.
func GetHomeDir() (string, error) {
	var homedir string

	// use the usr lib to get homedir then try $HOME if it does not work
	usr, err := user.Current()
	if err != nil {
		homedir = os.Getenv("HOME")
		if homedir == "" {
			return "", errors.New("Standard methods to locate user's home firectory failed. Please set $HOME and try again.")
		}
	} else {
		homedir = usr.HomeDir
	}

	return homedir, err
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
