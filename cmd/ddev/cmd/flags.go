package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// Flag is the structure for the flags, the json from the annotation is
// unmarshaled into this structure. For more information see also
// github.com/spf13/pflag/flag
type Flag struct {
	Name        string              // name as it appears on command line
	Shorthand   string              // one-letter abbreviated flag
	Usage       string              // help message
	Type        string              // type, defaults to bool
	DefValue    string              // default value (as text); for usage message
	NoOptDefVal string              // default value (as text); if the flag is on the command line without any options
	Annotations map[string][]string // used by cobra.Command bash autocomple code
}

// FlagsDefinition is an array of Flag holding all defined flags of a command.
type FlagsDefinition []Flag

// Defines the constants for the valid types which always should used in the
// source code.
const (
	FtBool           = "bool"
	FtBoolSlice      = "boolSlice"
	FtBytesHex       = "bytesHex"
	FtBytesBase64    = "bytesBase64"
	FtCount          = "count"
	FtDuration       = "duration"
	FtDurationSlice  = "durationSlice"
	FtFloat32        = "float32"
	FtFloat32Slice   = "float32Slice"
	FtFloat64        = "float64"
	FtFloat64Slice   = "float64Slice"
	FtInt            = "int"
	FtIntSlice       = "intSlice"
	FtInt8           = "int8"
	FtInt16          = "int16"
	FtInt32          = "int32"
	FtInt32Slice     = "int32Slice"
	FtInt64          = "int64"
	FtInt64Slice     = "int64Slice"
	FtIP             = "ip"
	FtIPSlice        = "ipSlice"
	FtIPMask         = "ipMask"
	FtIPNet          = "ipNet"
	FtString         = "string"
	FtStringArray    = "stringArray"
	FtStringSlice    = "stringSlice"
	FtStringToInt    = "stringToInt"
	FtStringToInt64  = "stringToInt64"
	FtStringToString = "stringToString"
	FtUint           = "uint"
	FtUintSlice      = "uintSlice"
	FtUint8          = "uint8"
	FtUint16         = "uint16"
	FtUint32         = "uint32"
	FtUint64         = "uint64"
	FtNotImplemented = "notimplemented" // is used for testing only
)

// ValidTypes defines the valid types, a value of true indicates it's
// implemented.
// To implement a new type add the required line to the switch statement in
// AssignToCommand and set it here to true, that's all. If a new type is
// added which is not defined here just add a new constant above and here.
var ValidTypes = map[string]bool{
	FtBool:           true,
	FtBoolSlice:      false,
	FtBytesHex:       false,
	FtBytesBase64:    false,
	FtCount:          false,
	FtDuration:       false,
	FtDurationSlice:  false,
	FtFloat32:        false,
	FtFloat32Slice:   false,
	FtFloat64:        false,
	FtFloat64Slice:   false,
	FtInt:            true,
	FtIntSlice:       false,
	FtInt8:           false,
	FtInt16:          false,
	FtInt32:          false,
	FtInt32Slice:     false,
	FtInt64:          false,
	FtInt64Slice:     false,
	FtIP:             false,
	FtIPSlice:        false,
	FtIPMask:         false,
	FtIPNet:          false,
	FtString:         true,
	FtStringArray:    false,
	FtStringSlice:    false,
	FtStringToInt:    false,
	FtStringToInt64:  false,
	FtStringToString: false,
	FtUint:           true,
	FtUintSlice:      false,
	FtUint8:          false,
	FtUint16:         false,
	FtUint32:         false,
	FtUint64:         false,
	FtNotImplemented: false,
}

// Flags is the main type used to access flags and methods.
type Flags struct {
	CommandName string
	Script      string
	Definition  FlagsDefinition
}

// Init initializes the Flags structure.
func (f *Flags) Init(commandName, script string) {
	f.CommandName = commandName
	f.Script = script
}

func validateFlag(flag *Flag) error {
	var errors string

	// Check shorthand is one letter only
	if len(flag.Shorthand) > 1 {
		errors += fmt.Sprintf("\n - shorthand '%s' for flag '%s' is more than one ASCII character", flag.Shorthand, flag.Name)
	}

	// Check usage is defined
	if flag.Usage == "" {
		errors += fmt.Sprintf("\n - no usage defined for flag '%s'", flag.Name)
	}

	// Check type is valid
	if flag.Type != "" {
		implemented, found := ValidTypes[flag.Type]

		if !found {
			errors += fmt.Sprintf("\n - type '%s' for flag '%s' is not known", flag.Type, flag.Name)
		} else if !implemented {
			errors += fmt.Sprintf("\n - type '%s' for flag '%s' is not implemented", flag.Type, flag.Name)
		}
	}

	return fmt.Errorf("%s", errors)
}

func (f *Flags) validateFlags(flags *FlagsDefinition) error {
	var errors string

	// Temporay vars to precheck for duplicated flags. It's still possible
	// other commands will introduce the same flags which is tested
	// afterwards by cobra.
	long := map[string]bool{}
	short := map[string]string{}

	for i := range *flags {
		flag := &(*flags)[i]

		// Check flag does not already exist
		if _, found := long[flag.Name]; found {
			errors += fmt.Sprintf("\n - flag '%s' already defined", flag.Name)
		} else {
			long[flag.Name] = true
		}

		// Check shorthand does not already exist
		if flagOfShorthand, found := short[flag.Shorthand]; found {
			errors += fmt.Sprintf("\n - shorthand '%s' is already defined flag '%s'", flag.Shorthand, flagOfShorthand)
		} else {
			short[flag.Shorthand] = flag.Name
		}

		// Check type and set default if empty
		if flag.Type == "" {
			flag.Type = FtBool
		}

		// Additional validations of the flag fields
		errors += validateFlag(flag).Error()
	}

	if errors != "" {
		return fmt.Errorf("The following problems were found in the flags definition of the command '%s' in '%s':%s", f.CommandName, f.Script, errors)
	}

	return nil
}

// LoadFromJSON imports the defs provided by the custom command as json into
// the flags structure.
func (f *Flags) LoadFromJSON(data string) error {
	if data == "" {
		return nil
	}

	var defs *FlagsDefinition
	var err error

	// Import the JSON to the FlagsDefinition structure and return in case of
	// error
	if err = json.Unmarshal([]byte(data), &defs); err != nil {
		return err
	}

	// Validate the user provided flags and return in case of error
	if err = f.validateFlags(defs); err != nil {
		return err
	}

	// Assign the data to the field
	f.Definition = *defs
	return nil
}

// AssignToCommand iterates the flags and assigns it to the provided command.
func (f *Flags) AssignToCommand(command *cobra.Command) error {
	for _, flag := range f.Definition {
		// Create the flag at the command
		switch flag.Type {
		case FtBool /*, ""*/ : // empty type defaults to bool
			command.Flags().BoolP(flag.Name, flag.Shorthand, false, flag.Usage)
		case FtInt:
			command.Flags().IntP(flag.Name, flag.Shorthand, 0, flag.Usage)
		case FtString:
			command.Flags().StringP(flag.Name, flag.Shorthand, "", flag.Usage)
		case FtUint:
			command.Flags().UintP(flag.Name, flag.Shorthand, 0, flag.Usage)
		default:
			continue // continue here, nothing to set for this flag
		}

		// Update default values and annotations
		newFlag := command.Flags().Lookup(flag.Name)

		if err := newFlag.Value.Set(flag.DefValue); err != nil {
			// Invalid default value was defined by the user, hide the flag
			// and return
			newFlag.Hidden = true
			return err
		}

		newFlag.DefValue = flag.DefValue
		newFlag.NoOptDefVal = flag.NoOptDefVal
		newFlag.Annotations = flag.Annotations
	}

	return nil
}
