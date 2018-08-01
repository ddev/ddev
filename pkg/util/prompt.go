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
	fullPrompt := fmt.Sprintf("%s (%s):", prompt, defaultValue)
	fmt.Print(fullPrompt + ": ")
	return GetInput(defaultValue)
}
