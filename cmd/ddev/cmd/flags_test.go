package cmd

import (
	asrt "github.com/stretchr/testify/assert"
	"testing"
)

const (
	commandName = "command"
	script      = "script"
)

func getSubject() Flags {
	var subject Flags
	subject.Init(commandName, script)
	return subject
}

func TestInit(t *testing.T) {
	assert := asrt.New(t)
	subject := getSubject()

	assert.Exactly(t, commandName, subject.CommandName)
	assert.Exactly(t, script, subject.Script)
}

// TestFlags does basic checks to make sure custom commands work OK.
func TestFlags(t *testing.T) {
	assert := asrt.New(t)
	subject := getSubject()

	//
	err := subject.LoadFromJson("[]")
	assert.NoError(err)
}

/*
[{"Long":"test-1","Short":"t","Usage":"Usage of test 1"},{"Long":"test-1","Usage":"Test duplicate"}]
## Flags: [{"Long":"test-1","Short":"t","Usage":"Usage of test 1"},{"Long":"test-2","Short":"t","Usage":"Usage of test 2 with existing shorthand"}]
## Flags: {[{"Long":"test-1","Short":"t","Usage":"Usage of test 1"}]
## Flags: [{"Long":"test-1","Short":"t1","Usage":"Usage of test 1"}]
## Flags: [{"Long":"test-1","Short":"t","Usage":""},{"Long":"test-2"}]


	// Test possible user errors are handled properly for Flags
	c := "testflagsinvalidjson"
	args := []string{c}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "Failed to run ddev %s", c)
	assert.Contains(out, fmt.Sprintf("command '%s' contains an invalid flags definition", c))

	c = "testflagsmissingusage"
	args = []string{c}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "Failed to run ddev %s", c)
	assert.Contains(out, fmt.Sprintf("No usage defined for flag '%s' of command '%s', skipping add flag defined in ", "test-1", c))

	c = "testflagsduplicate"
	args = []string{c}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "Failed to run ddev %s", c)
	assert.Contains(out, fmt.Sprintf("Flag '%s' already defined for command '%s', skipping add flag defined in ", "test-1", c))

	c = "testflagsduplicateshorthand"
	args = []string{c}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "Failed to run ddev %s", c)
	assert.Contains(out, fmt.Sprintf("Shorthand '%s' already defined for command '%s', skipping add flag defined in ", "t", c))

	c = "testflagslongshorthand"
	args = []string{c}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err, "Failed to run ddev %s", c)
	assert.Contains(out, fmt.Sprintf("Shorthand '%s' with more than one ASCII character defined for command '%s', skipping add flag defined in ", "t1", c))

*/
