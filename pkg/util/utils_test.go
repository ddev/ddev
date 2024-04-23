package util_test

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
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
	restoreOutput := util.CaptureUserOut()
	scanner := bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)
	result := util.GetInput("nodefault")
	assert.EqualValues(input, result)
	_ = restoreOutput()

	// Try Prompt() with a default value which is overridden
	input = "InputIWantToSee"
	restoreOutput = util.CaptureUserOut()
	scanner = bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)
	result = util.Prompt("nodefault", "expected default")
	assert.EqualValues(input, result)
	_ = restoreOutput()

	// Try Prompt() with a default value but don't provide a response
	input = ""
	restoreOutput = util.CaptureUserOut()
	scanner = bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)
	result = util.Prompt("nodefault", "expected default")
	assert.EqualValues("expected default", result)
	_ = restoreOutput()
	println() // Just lets goland find the PASS or FAIL
}

// TestCaptureUserOut ensures capturing of stdout works as expected.
func TestCaptureUserOut(t *testing.T) {
	assert := asrt.New(t)
	restoreOutput := util.CaptureUserOut()
	text := util.RandString(128)
	output.UserOut.Println(text)
	out := restoreOutput()

	assert.Contains(out, text)
}

// TestCaptureStdOut ensures capturing of stdout works as expected.
func TestCaptureStdOut(t *testing.T) {
	assert := asrt.New(t)
	restoreOutput := util.CaptureStdOut()
	text := util.RandString(128)
	fmt.Println(text)
	out := restoreOutput()

	assert.Contains(out, text)
}

// TestCaptureOutputToFile tests CaptureOutputToFile.
func TestCaptureOutputToFile(t *testing.T) {
	assert := asrt.New(t)
	restoreOutput, err := util.CaptureOutputToFile()
	assert.NoError(err)
	text := util.RandString(128)
	fmt.Println("randstring-println=" + text)
	c := exec.Command("sh", "-c", fmt.Sprintf("echo randstring-stdout=%s; echo 1>&2 randstring-stderr=%s", text, text))
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err = c.Start()
	assert.NoError(err)
	err = c.Wait()
	assert.NoError(err)

	out := restoreOutput()

	assert.Contains(out, "randstring-println="+text)
	assert.Contains(out, "randstring-stdout="+text)
	assert.Contains(out, "randstring-stderr="+text)
}

// TestConfirmTo ensures that the confirmation prompt works as expected.
func TestConfirmTo(t *testing.T) {
	assert := asrt.New(t)

	// Unset the env var, then reset after the test
	noninteractiveEnv := "DDEV_NONINTERACTIVE"
	defer os.Setenv(noninteractiveEnv, os.Getenv(noninteractiveEnv))
	err := os.Unsetenv(noninteractiveEnv)
	if err != nil {
		t.Fatal(err.Error())
	}

	// test a given input against a default value
	testInput := func(input string, defaultTo bool, expected bool) {
		text := util.RandString(32)
		scanner := bufio.NewScanner(strings.NewReader(input))
		util.SetInputScanner(scanner)

		getOutput := util.CaptureStdOut()
		resp := util.ConfirmTo(text, defaultTo)
		if expected {
			assert.True(resp)
		} else {
			assert.False(resp)
		}
		assert.Contains(getOutput(), text)
	}

	yesses := []string{"YES", "Yes", "yes", "Y", "y"}
	for _, y := range yesses {
		testInput(y, true, true)
		testInput(y, false, true)
	}

	nos := []string{"NO", "No", "no", "N", "n"}
	for _, n := range nos {
		testInput(n, true, false)
		testInput(n, false, false)
	}

	// Test that junk answers (not yes, no, or <enter>) eventually return a false
	junkText := "a\nb\na\nb\na\nb\na\nb\na\nb\n"
	testInput(junkText, true, false)
	testInput(junkText, false, false)

	// Test that <enter> returns the defaultTo value
	enter := "\n"
	testInput(enter, true, true)
	testInput(enter, false, false)
}

// TestSliceToUniqueSlice tests SliceToUniqueSlice
func TestSliceToUniqueSlice(t *testing.T) {
	assert := asrt.New(t)

	testBedSources := [][]string{
		{"1", "2", "3", "2", "3", "1"},
		{"99", "98", "97", "99", "98", "97", "1", "2", "3"},
	}

	testBedExpectations := [][]string{
		{"1", "2", "3"},
		{"99", "98", "97", "1", "2", "3"},
	}

	for i := 0; i < len(testBedSources); i++ {
		res := util.SliceToUniqueSlice(&testBedSources[i])
		assert.Equal(testBedExpectations[i], res)
	}
}

// TestArrayToReadableOutput tests ArrayToReadableOutput
func TestArrayToReadableOutput(t *testing.T) {
	assert := asrt.New(t)

	testSource := []string{
		"file1.conf",
		"file2.conf",
	}
	expectation := `[
	file1.conf
	file2.conf
]`
	res, _ := util.ArrayToReadableOutput(testSource)
	assert.Equal(expectation, res)

	_, err := util.ArrayToReadableOutput([]string{})
	assert.EqualErrorf(err, "empty slice", "Expected error when passing an empty slice")
}
