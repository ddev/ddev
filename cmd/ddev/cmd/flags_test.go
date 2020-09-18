package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	asrt "github.com/stretchr/testify/assert"
)

const (
	commandName = "command"
	script      = "script"
)

// getSubject returns a new, initialized Flags struct.
func getSubject() Flags {
	var subject Flags
	subject.Init(commandName, script)
	return subject
}

// TestUnitCmdFlagsInit tests the Init method.
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
	assert.EqualValues("test", subject.Definition[0].Name)
	assert.EqualValues("", subject.Definition[0].Shorthand)
	assert.EqualValues("Usage of test", subject.Definition[0].Usage)
	assert.EqualValues(FtBool, subject.Definition[0].Type)
	assert.EqualValues("false", subject.Definition[0].DefValue)
	assert.EqualValues("true", subject.Definition[0].NoOptDefVal)
	assert.Empty(subject.Definition[0].Annotations)

	// Full
	assert.NoError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":"Usage of test 1","Type":"string","DefValue":"true","NoOptDefVal":"true","Annotations":{"test-1":["test-1-1","test-1-2"]}},{"Name":"test-2","Usage":"Usage of test 2","Type":"bool","DefValue":"true","NoOptDefVal":"true","Annotations":{"test-2":["test-2-1","test-2-2"]}}]`))
	assert.EqualValues("test-1", subject.Definition[0].Name)
	assert.EqualValues("t", subject.Definition[0].Shorthand)
	assert.EqualValues("Usage of test 1", subject.Definition[0].Usage)
	assert.EqualValues(FtString, subject.Definition[0].Type)
	assert.EqualValues("true", subject.Definition[0].DefValue)
	assert.EqualValues("true", subject.Definition[0].NoOptDefVal)
	assert.EqualValues(map[string][]string{"test-1": {"test-1-1", "test-1-2"}}, subject.Definition[0].Annotations)
	assert.EqualValues("test-2", subject.Definition[1].Name)
	assert.EqualValues("", subject.Definition[1].Shorthand)
	assert.EqualValues("Usage of test 2", subject.Definition[1].Usage)
	assert.EqualValues(FtBool, subject.Definition[1].Type)
	assert.EqualValues("true", subject.Definition[1].DefValue)
	assert.EqualValues("true", subject.Definition[1].NoOptDefVal)
	assert.EqualValues(map[string][]string{"test-2": {"test-2-1", "test-2-2"}}, subject.Definition[1].Annotations)

	// Duplicate flag
	assert.EqualError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":"Usage of test 1"},{"Name":"test-1","Usage":"Test duplicate"}]`),
		"The following problems were found in the flags definition of the command 'command' in 'script':\n * for flag 'test-1':\n   - flag 'test-1' already defined")

	// Duplicate shorthand
	assert.EqualError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":"Usage of test 1"},{"Name":"test-2","Shorthand":"t","Usage":"Usage of test 2 with existing shorthand"}]`),
		"The following problems were found in the flags definition of the command 'command' in 'script':\n * for flag 'test-2':\n   - shorthand 't' is already defined for flag 'test-1'")

	// Invalid shorthand
	assert.EqualError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t1","Usage":"Usage of test 1"}]`),
		"The following problems were found in the flags definition of the command 'command' in 'script':\n * for flag 'test-1':\n   - shorthand 't1' is more than one ASCII character")

	// Empty usage in multiple commands
	assert.EqualError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":""},{"Name":"test-2"}]`),
		"The following problems were found in the flags definition of the command 'command' in 'script':\n * for flag 'test-1':\n   - no usage defined\n * for flag 'test-2':\n   - no usage defined")

	// Invalid and not implemented type
	assert.EqualError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":"Usage of test 1","Type":"_invalid_"},{"Name":"test-2","Usage":"Usage of test 2","Type":"_test1_"}]`),
		"The following problems were found in the flags definition of the command 'command' in 'script':\n * for flag 'test-1':\n   - type '_invalid_' is not known\n * for flag 'test-2':\n   - type '_test1_' is not implemented")
	assert.PanicsWithValue("Error implementation of DefValue validation missing for type '_test2_'", func() {
		assert.NoError(subject.LoadFromJSON(`[{"Name":"test-1","Usage":"Usage of test 1","Type":"_test2_"}]`))
	})
}

// getCommand returns a new cobra Command.
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

	var subject Flags
	var c cobra.Command
	var f *pflag.Flag

	// No flags
	subject = getSubject()
	c = getCommand()
	assert.NoError(subject.AssignToCommand(&c))

	// Minimal
	assert.NoError(subject.LoadFromJSON(`[{"Name":"test","Usage":"Usage of test"}]`))
	assert.NoError(subject.AssignToCommand(&c))

	f = c.Flags().Lookup("test")
	assert.NotEmpty(f)
	assert.EqualValues("test", f.Name)
	assert.EqualValues("", f.Shorthand)
	assert.EqualValues("Usage of test", f.Usage)
	assert.EqualValues(FtBool, f.Value.Type())
	assert.EqualValues("false", f.DefValue)
	assert.EqualValues("true", f.NoOptDefVal)
	assert.Empty(f.Annotations)

	// Full
	subject = getSubject()
	c = getCommand()
	assert.NoError(subject.LoadFromJSON(`[{"Name":"test-1","Shorthand":"t","Usage":"Usage of test 1","Type":"bool","DefValue":"true","NoOptDefVal":"true","Annotations":{"test-1":["test-1-1","test-1-2"]}},{"Name":"test-2","Usage":"Usage of test 2","Type":"string","DefValue":"DefValue","NoOptDefVal":"NoOptDefVal"},{"Name":"test-3","Usage":"Usage of test 3","Type":"int","DefValue":"1","NoOptDefVal":"-1"},{"Name":"test-4","Usage":"Usage of test 4","Type":"uint","DefValue":"1","NoOptDefVal":"2"}]`))
	assert.NoError(subject.AssignToCommand(&c))

	f = c.Flags().Lookup("test-1")
	assert.NotEmpty(f)
	assert.EqualValues("test-1", f.Name)
	assert.EqualValues("t", f.Shorthand)
	assert.EqualValues("Usage of test 1", f.Usage)
	assert.EqualValues(FtBool, f.Value.Type())
	assert.EqualValues("true", f.DefValue)
	assert.EqualValues("true", f.NoOptDefVal)
	assert.EqualValues(map[string][]string{"test-1": {"test-1-1", "test-1-2"}}, f.Annotations)

	f = c.Flags().Lookup("test-2")
	assert.EqualValues("test-2", f.Name)
	assert.EqualValues("", f.Shorthand)
	assert.EqualValues("Usage of test 2", f.Usage)
	assert.EqualValues(FtString, f.Value.Type())
	assert.EqualValues("DefValue", f.DefValue)
	assert.EqualValues("NoOptDefVal", f.NoOptDefVal)
	assert.Empty(f.Annotations)

	f = c.Flags().Lookup("test-3")
	assert.EqualValues("test-3", f.Name)
	assert.EqualValues("", f.Shorthand)
	assert.EqualValues("Usage of test 3", f.Usage)
	assert.EqualValues(FtInt, f.Value.Type())
	assert.EqualValues("1", f.DefValue)
	assert.EqualValues("-1", f.NoOptDefVal)
	assert.Empty(f.Annotations)

	f = c.Flags().Lookup("test-4")
	assert.EqualValues("test-4", f.Name)
	assert.EqualValues("", f.Shorthand)
	assert.EqualValues("Usage of test 4", f.Usage)
	assert.EqualValues(FtUint, f.Value.Type())
	assert.EqualValues("1", f.DefValue)
	assert.EqualValues("2", f.NoOptDefVal)
	assert.Empty(f.Annotations)

	// Not fully implemented type
	subject = getSubject()
	c = getCommand()
	assert.NoError(subject.LoadFromJSON(`[{"Name":"test","Usage":"Usage of test","Type":"_test3_"}]`))
	assert.PanicsWithValue("Error implementation missing for type '_test3_'", func() {
		assert.NoError(subject.AssignToCommand(&c))
	})

	// Invalid DefValue
	subject = getSubject()
	c = getCommand()
	assert.NoError(subject.LoadFromJSON(`[{"Name":"test-1","Usage":"Usage of test 1","Type":"bool","DefValue":"no-bool-value"},{"Name":"test-2","Usage":"Usage of test 2","Type":"int","DefValue":"no-int-value"}]`))
	assert.EqualError(subject.AssignToCommand(&c),
		"The following problems were found while assigning the flags to the command 'command' in 'script':\n - error 'strconv.ParseBool: parsing \"no-bool-value\": invalid syntax' while set value of flag 'test-1'\n - error 'strconv.ParseInt: parsing \"no-int-value\": invalid syntax' while set value of flag 'test-2'")
}
