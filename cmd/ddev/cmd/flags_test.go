package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	asrt "github.com/stretchr/testify/assert"
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

func TestUnitCmdFlagsInit(t *testing.T) {
	assert := asrt.New(t)
	subject := getSubject()

	assert.Exactly(commandName, subject.CommandName)
	assert.Exactly(script, subject.Script)
}

// TestCmdFlagsLoadFromJSON checks LoadFromJSON works correctly and handles
// user errors.
func TestUnitCmdFlagsLoadFromJSON(t *testing.T) {
	assert := asrt.New(t)
	subject := getSubject()

	// No data
	assert.NoError(subject.LoadFromJSON(``))

	// Invalid JSON
	assert.Error(subject.LoadFromJSON(`this is no valid JSON`))

	// Empty array
	assert.NoError(subject.LoadFromJSON(`[]`))

	// Minimal
	assert.NoError(subject.LoadFromJSON(`[{"Name":"test","Usage":"Usage of test"}]`))
	assert.Exactly("test", subject.Definition[0].Name)
	assert.Exactly("", subject.Definition[0].Shorthand)
	assert.Exactly("Usage of test", subject.Definition[0].Usage)
	assert.Exactly("", subject.Definition[0].Type)
	assert.Exactly("", subject.Definition[0].DefValue)
	assert.Exactly("", subject.Definition[0].NoOptDefVal)
	assert.Empty(subject.Definition[0].Annotations)

	// Full
	assert.NoError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":"Usage of test 1","Type":"bool","DefValue":"true","NoOptDefVal":"true","Annotations":{"test-1":["test-1-1","test-1-2"]}},{"Name":"test-2","Usage":"Usage of test 2","Type":"bool","DefValue":"true","NoOptDefVal":"true","Annotations":{"test-2":["test-2-1","test-2-2"]}}]`))
	assert.Exactly("test-1", subject.Definition[0].Name)
	assert.Exactly("t", subject.Definition[0].Shorthand)
	assert.Exactly("Usage of test 1", subject.Definition[0].Usage)
	assert.Exactly("bool", subject.Definition[0].Type)
	assert.Exactly("true", subject.Definition[0].DefValue)
	assert.Exactly("true", subject.Definition[0].NoOptDefVal)
	assert.Exactly(map[string][]string{"test-1": {"test-1-1", "test-1-2"}}, subject.Definition[0].Annotations)
	assert.Exactly("test-2", subject.Definition[1].Name)
	assert.Exactly("", subject.Definition[1].Shorthand)
	assert.Exactly("Usage of test 2", subject.Definition[1].Usage)
	assert.Exactly("bool", subject.Definition[1].Type)
	assert.Exactly("true", subject.Definition[1].DefValue)
	assert.Exactly("true", subject.Definition[1].NoOptDefVal)
	assert.Exactly(map[string][]string{"test-2": {"test-2-1", "test-2-2"}}, subject.Definition[1].Annotations)

	// Duplicate flag
	assert.EqualError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":"Usage of test 1"},{"Name":"test-1","Usage":"Test duplicate"}]`),
		"The following problems were found in the flags definition of the command 'command' in 'script':\n - flag 'test-1' already defined")

	// Duplicate shorthand
	assert.EqualError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":"Usage of test 1"},{"Name":"test-2","Shorthand":"t","Usage":"Usage of test 2 with existing shorthand"}]`),
		"The following problems were found in the flags definition of the command 'command' in 'script':\n - shorthand 't' is already defined flag 'test-1'")

	// Invalid shorthand
	assert.EqualError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t1","Usage":"Usage of test 1"}]`),
		"The following problems were found in the flags definition of the command 'command' in 'script':\n - shorthand 't1' for flag 'test-1' is more than one ASCII character")

	// Empty usage in multiple commands
	assert.EqualError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":""},{"Name":"test-2"}]`),
		"The following problems were found in the flags definition of the command 'command' in 'script':\n - no usage defined for flag 'test-1'\n - no usage defined for flag 'test-2'")

	// Invalid and not implemented type
	assert.EqualError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":"Usage of test 1","Type":"invalid"},{"Name":"test-2","Usage":"Usage of test 2","Type":"notimplemented"}]`),
		"The following problems were found in the flags definition of the command 'command' in 'script':\n - type 'invalid' for flag 'test-1' is not known\n - type 'notimplemented' for flag 'test-2' is not implemented")
}

func getCommand() cobra.Command {
	return cobra.Command{
		Use:     "usage of command",
		Short:   "short description",
		Example: "example",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}

}

// TestUnitCmdFlagsAssignToCommand checks AssignToCommand works correctly and
// handles user errors.
func TestUnitCmdFlagsAssignToCommand(t *testing.T) {
	assert := asrt.New(t)
	subject := getSubject()

	var c cobra.Command

	// No flags
	c = getCommand()
	assert.NoError(subject.AssignToCommand(&c))

	// No flags
	c = getCommand()
	assert.NoError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":"Usage of test 1","Type":"bool","DefValue":"true","NoOptDefVal":"true","Annotations":{"test-1":["test-1-1","test-1-2"]}},{"Name":"test-2","Usage":"Usage of test 2","Type":"bool","DefValue":"true","NoOptDefVal":"true","Annotations":{"test-2":["test-2-1","test-2-2"]}}]`))
	assert.NoError(subject.AssignToCommand(&c))
}
