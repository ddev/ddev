package utils

import (
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
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
	log.WithFields(log.Fields{
		"Command": command + " " + strings.Join(args[:], " "),
	}).Info("Running Command")

	out, err := exec.Command(
		command, args...,
	).CombinedOutput()

	log.WithFields(log.Fields{
		"Result": string(out),
	}).Debug("Command Result")

	return string(out), err
}

// RunCommandPipe runs a command on the host system while piping output to stderr and stdout.
func RunCommandPipe(command string, args []string) error {
	log.WithFields(log.Fields{
		"Command": command + " " + strings.Join(args[:], " "),
	}).Info("Running Command")

	proc := exec.Command(command, args...)
	proc.Stdout = os.Stdout
	proc.Stdin = os.Stdin
	proc.Stderr = os.Stderr

	return proc.Run()
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

// DownloadFile retreives a file.
func DownloadFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
