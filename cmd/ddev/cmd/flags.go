package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
)

// The array structure for the flags, the json from the annotation is
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

type FlagsDefinition []Flag

// Defines the constants for the valid types which always should used in the
// source code.
const (
	FT_BOOL             = "bool"
	FT_BOOL_SLICE       = "boolSlice"
	FT_BYTES_HEX        = "bytesHex"
	FT_BYTES_BASE64     = "bytesBase64"
	FT_COUNT            = "count"
	FT_DURATION         = "duration"
	FT_DURATION_SLICE   = "durationSlice"
	FT_FLOAT32          = "float32"
	FT_FLOAT32_SLICE    = "float32Slice"
	FT_FLOAT64          = "float64"
	FT_FLOAT64_SLICE    = "float64Slice"
	FT_INT              = "int"
	FT_INT_SLICE        = "intSlice"
	FT_INT8             = "int8"
	FT_INT16            = "int16"
	FT_INT32            = "int32"
	FT_INT32_SLICE      = "int32Slice"
	FT_INT64            = "int64"
	FT_INT64_SLICE      = "int64Slice"
	FT_IP               = "ip"
	FT_IP_SLICE         = "ipSlice"
	FT_IP_MASK          = "ipMask"
	FT_IP_NET           = "ipNet"
	FT_STRING           = "string"
	FT_STRING_ARRAY     = "stringArray"
	FT_STRING_SLICE     = "stringSlice"
	FT_STRING_TO_INT    = "stringToInt"
	FT_STRING_TO_INT64  = "stringToInt64"
	FT_STRING_TO_STRING = "stringToString"
	FT_UINT             = "uint"
	FT_UINT_Slice       = "uintSlice"
	FT_UINT8            = "uint8"
	FT_UINT16           = "uint16"
	FT_UINT32           = "uint32"
	FT_UINT64           = "uint64"
)

// Defines the valid types, a value of true indicates it's implemented.
// To implement a new type add the required line to the switch statement in
// AssignToCommand and set it here to true, that's all. If a new type is
// added which is not defined here just add a new constant above and here.
var ValidTypes = map[string]bool{
	FT_BOOL:             true,
	FT_BOOL_SLICE:       false,
	FT_BYTES_HEX:        false,
	FT_BYTES_BASE64:     false,
	FT_COUNT:            false,
	FT_DURATION:         false,
	FT_DURATION_SLICE:   false,
	FT_FLOAT32:          false,
	FT_FLOAT32_SLICE:    false,
	FT_FLOAT64:          false,
	FT_FLOAT64_SLICE:    false,
	FT_INT:              true,
	FT_INT_SLICE:        false,
	FT_INT8:             false,
	FT_INT16:            false,
	FT_INT32:            false,
	FT_INT32_SLICE:      false,
	FT_INT64:            false,
	FT_INT64_SLICE:      false,
	FT_IP:               false,
	FT_IP_SLICE:         false,
	FT_IP_MASK:          false,
	FT_IP_NET:           false,
	FT_STRING:           true,
	FT_STRING_ARRAY:     false,
	FT_STRING_SLICE:     false,
	FT_STRING_TO_INT:    false,
	FT_STRING_TO_INT64:  false,
	FT_STRING_TO_STRING: false,
	FT_UINT:             true,
	FT_UINT_Slice:       false,
	FT_UINT8:            false,
	FT_UINT16:           false,
	FT_UINT32:           false,
	FT_UINT64:           false,
}

// The main type used to access flags and methods
type Flags struct {
	CommandName string
	Script      string
	Definition  FlagsDefinition
}

func (f *Flags) Init(commandName, script string) {
	f.CommandName = commandName
	f.Script = script
}

func (f *Flags) validateFlags(flags FlagsDefinition) error {
	var errors string
	long := map[string]bool{}
	short := map[string]string{}

	for _, flag := range flags {
		// Check flag does not already exist
		if _, found := long[flag.Name]; found {
			errors += fmt.Sprintf("\n - flag '%s' already defined", flag.Name)
		} else {
			long[flag.Name] = true
		}

		// Check shorthand is one letter only
		if len(flag.Shorthand) > 1 {
			errors += fmt.Sprintf("\n - shorthand '%s' for flag '%s' is more than one ASCII character", flag.Shorthand, flag.Name)
		} else {
			// Check shorthand does not already exist
			if flagOfShorthand, found := short[flag.Shorthand]; found {
				errors += fmt.Sprintf("\n - shorthand '%s' is already defined flag '%s'", flag.Shorthand, flagOfShorthand)
			} else {
				short[flag.Shorthand] = flag.Name
			}
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
	}

	if errors != "" {
		return fmt.Errorf("The following problems were found in the flags definition of the command '%s' in '%s':%s", f.CommandName, f.Script, errors)
	}

	return nil
}

// Imports the defs provided by the custom command as json into the flags
// structure.
func (f *Flags) LoadFromJson(data string) error {
	if data == "" {
		return nil
	}

	var defs FlagsDefinition
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
	f.Definition = defs
	return nil
}

// Iterates the flags and assigns it to the provided command.
func (f *Flags) AssignToCommand(command *cobra.Command) error {
	for _, flag := range f.Definition {
		// Create the flag at the command
		switch flag.Type {
		case FT_BOOL, "": // no type defaults to bool
			command.Flags().BoolP(flag.Name, flag.Shorthand, false, flag.Usage)
		case FT_INT:
			command.Flags().IntP(flag.Name, flag.Shorthand, 0, flag.Usage)
		case FT_STRING:
			command.Flags().StringP(flag.Name, flag.Shorthand, "", flag.Usage)
		case FT_UINT:
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
