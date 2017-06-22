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
