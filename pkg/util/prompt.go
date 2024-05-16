package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var inputScanner = bufio.NewScanner(os.Stdin)

// SetInputScanner allows you to override the default input scanner with your own.
func SetInputScanner(scanner *bufio.Scanner) {
	inputScanner = scanner
}

// GetInput reads input from an input buffer and returns the result as a string.
func GetInput(defaultValue string) string {
	inputScanner.Scan()
	input := inputScanner.Text()

	// If the value from the input buffer is blank, then use the default instead.
	value := strings.TrimSpace(input)
	if value == "" {
		value = defaultValue
	}

	return value
}

// Prompt gets input with a prompt and returns the input
func Prompt(prompt string, defaultValue string) string {
	fullPrompt := fmt.Sprintf("%s (%s)", prompt, defaultValue)
	fmt.Print(fullPrompt + ": ")
	return GetInput(defaultValue)
}

// Confirm handles the asking and interpreting of a basic yes/no question.
// If DDEV_NONINTERACTIVE is set, Confirm() returns true. The prompt will be
// presented at most three times before returning false.
// Defaults to true if left blank.
func Confirm(prompt string) bool {
	return ConfirmTo(prompt, true)
}

// ConfirmTo handles the asking and interpreting of a basic yes/no question.
// If DDEV_NONINTERACTIVE is set, Confirm() returns true.
// If response is blank, the defaultTo value is returned.
// If response is invalid, the prompt will be presented at most three times
// before returning false.
func ConfirmTo(prompt string, defaultTo bool) bool {
	if len(os.Getenv("DDEV_NONINTERACTIVE")) > 0 {
		return true
	}

	var promptOptions string
	var promptDefaultValue string

	if defaultTo {
		promptOptions = "Y/n"
		promptDefaultValue = "yes"
	} else {
		promptOptions = "y/N"
		promptDefaultValue = "no"
	}

	for i := 0; i < 3; i++ {
		resp := strings.ToLower(Prompt(fmt.Sprintf("%s [%s]", prompt, promptOptions), promptDefaultValue))

		if resp == "yes" || resp == "y" {
			return true
		}

		if resp == "no" || resp == "n" {
			return false
		}
	}

	return false
}
