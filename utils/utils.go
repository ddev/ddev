package utils

import (
	"io"
	"net/http"
	"os"

	"os/exec"
)

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
