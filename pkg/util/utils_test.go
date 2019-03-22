package util_test

import (
	"bufio"
	"fmt"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
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

// TestConfirm ensures that the confirmation prompt works as expected.
func TestConfirm(t *testing.T) {
	assert := asrt.New(t)

	// Unset the env var, then reset after the test
	noninteractiveEnv := "DRUD_NONINTERACTIVE"
	defer os.Setenv(noninteractiveEnv, os.Getenv(noninteractiveEnv))
	err := os.Unsetenv(noninteractiveEnv)
	if err != nil {
		t.Fatal(err.Error())
	}

	yesses := []string{"YES", "Yes", "yes", "Y", "y"}
	for _, y := range yesses {
		text := util.RandString(32)
		scanner := bufio.NewScanner(strings.NewReader(y))
		util.SetInputScanner(scanner)

		getOutput := util.CaptureStdOut()
		resp := util.Confirm(text)
		assert.True(resp)
		assert.Contains(getOutput(), text)
	}

	nos := []string{"NO", "No", "no", "N", "n"}
	for _, n := range nos {
		text := util.RandString(32)
		scanner := bufio.NewScanner(strings.NewReader(n))
		util.SetInputScanner(scanner)

		getOutput := util.CaptureStdOut()
		resp := util.Confirm(text)
		assert.False(resp)
		assert.Contains(getOutput(), text)
	}

	// Test that junk answers (not yes, no, or <enter>) eventually return a false
	scanner := bufio.NewScanner(strings.NewReader("a\nb\na\nb\na\nb\na\nb\na\nb\n"))
	util.SetInputScanner(scanner)
	getOutput := util.CaptureStdOut()
	text := util.RandString(32)
	resp := util.Confirm(text)
	assert.False(resp)
	assert.Contains(getOutput(), text)

	// Test that <enter> returns true
	scanner = bufio.NewScanner(strings.NewReader("\n"))
	util.SetInputScanner(scanner)
	getOutput = util.CaptureStdOut()
	text = util.RandString(32)
	resp = util.Confirm(text)
	assert.True(resp)
	assert.Contains(getOutput(), text)
}

// TestGetFreePort checks util.GetFreePort() to make sure it respects
// ports reserved in DdevGlobalConfig.UsedHostPorts
// and that the port can actually be bound.
func TestGetFreePort(t *testing.T) {
	assert := asrt.New(t)

	dockerIP, err := dockerutil.GetDockerIP()
	require.NoError(t, err)

	// Find out a starting port the OS is likely to give us.
	startPort, err := util.GetFreePort(dockerIP)
	require.NoError(t, err)

	// Put 100 used ports in the UsedHostPorts
	i, err := strconv.Atoi(startPort)
	i = i + 1
	max := i + 100
	require.NoError(t, err)
	for ; i < max; i++ {
		globalconfig.DdevGlobalConfig.UsedHostPorts = append(globalconfig.DdevGlobalConfig.UsedHostPorts, strconv.Itoa(i))
	}

	for try := 0; try < 5; try++ {
		port, err := util.GetFreePort(dockerIP)
		require.NoError(t, err)
		assert.NotContains(globalconfig.DdevGlobalConfig.UsedHostPorts, port)

		// Make sure we can actually use the port.
		dockerCommand := []string{"run", "--rm", "-p" + dockerIP + ":" + port + ":" + port, "busybox:latest"}
		_, err = exec.RunCommand("docker", dockerCommand)

		assert.NoError(err, "failed to 'docker %v': %v", dockerCommand, err)
	}
}
