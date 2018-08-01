package util_test

import (
	"bufio"
	"fmt"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

// TestRandString ensures that RandString only generates string of the correct value and characters.
func TestRandString(t *testing.T) {
	assert := asrt.New(t)
	stringLengths := []int{2, 4, 8, 16, 23, 47}

	for _, stringLength := range stringLengths {
		testString := util.RandString(stringLength)
		assert.Equal(len(testString), stringLength, fmt.Sprintf("Generated string is of length %d", stringLengths))
	}
	lb := "a"
	util.SetLetterBytes(lb)
	testString := util.RandString(1)
	assert.Equal(testString, lb)
}

// TestGetInput tests GetInput and Prompt()
func TestGetInput(t *testing.T) {
	assert := asrt.New(t)

	// Try basic GetInput
	input := "InputIWantToSee"
	restoreOutput := testcommon.CaptureUserOut()
	scanner := bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)
	result := util.GetInput("nodefault")
	assert.EqualValues(input, result)
	_ = restoreOutput()

	// Try Prompt() with a default value which is overridden
	input = "InputIWantToSee"
	restoreOutput = testcommon.CaptureUserOut()
	scanner = bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)
	result = util.Prompt("nodefault", "expected default")
	assert.EqualValues(input, result)
	_ = restoreOutput()

	// Try Prompt() with a default value but don't provide a response
	input = ""
	restoreOutput = testcommon.CaptureUserOut()
	scanner = bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)
	result = util.Prompt("nodefault", "expected default")
	assert.EqualValues("expected default", result)
	_ = restoreOutput()
	println() // Just lets goland find the PASS or FAIL
}
