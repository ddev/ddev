package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// Internal types to allow to provide methods.
type nameValue string
type shorthandValue string
type usageValue string
type typeValue string
type defValueValue string
type noOptDefValValue string
type annotationsValue map[string][]string

// Flag is the structure for the flags, the json from the annotation is
// unmarshaled into this structure. For more information see also
// github.com/spf13/pflag/flag.
type Flag struct {
	Name        nameValue        // name as it appears on command line
	Shorthand   shorthandValue   // one-letter abbreviated flag
	Usage       usageValue       // help message
	Type        typeValue        // type, defaults to bool
	DefValue    defValueValue    // default value (as text); for usage message
	NoOptDefVal noOptDefValValue // default value (as text); if the flag is on the command line without any options
	Annotations annotationsValue // used by cobra.Command bash autocomple code
}

// FlagsDefinition is an array of Flag holding all defined flags of a command.
type FlagsDefinition []Flag

// Defines the constants for the valid types which always should used in the
// source code.
const (
	ftTest1          = "_test1_" // used for testing only
	ftTest2          = "_test2_" // used for testing only
	ftTest3          = "_test3_" // used for testing only
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
)

// ValidTypes defines the valid types, a value of true indicates it's
// implemented.
// To implement a new type add the required line to the switch statement in
// AssignToCommand and set it here to true, that's all. If a new type is
// added which is not defined here just add a new constant above and here.
var ValidTypes = map[typeValue]bool{
	ftTest1:          false, // used for testing only
	ftTest2:          true,  // used for testing only
	ftTest3:          true,  // used for testing only
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
}

// Flags is the main type used to access flags and methods.
type Flags struct {
	CommandName string
	Script      string
	Definition  FlagsDefinition
}

// formatErrorItem returns a formatted message indented by the number of
// `level` and prefixed by a `prefix`.
func formatErrorItem(level int, prefix string, format string, a ...interface{}) error {
	return fmt.Errorf("%s %s %s", strings.Repeat(" ", (level-1)*2), prefix, fmt.Sprintf(format, a...))
}

// extractError returns the error message of an error `err` prefixed by a new
// line or an empty string if no error was given.
func extractError(err error) string {
	if err != nil {
		return "\n" + strings.Trim(err.Error(), "\n")
	}

	return ""
}

// validate checks a nameValue for uniqueness.
func (v *nameValue) validate(longOptions *map[nameValue]bool) error {
	// Check flag does not already exist
	if _, found := (*longOptions)[*v]; found {
		return formatErrorItem(2, "-", "flag '%s' already defined", *v)
	}

	(*longOptions)[*v] = true

	return nil
}

// validate checks a shorthandValue for uniqueness and containing one ASCII
// character only.
func (v *shorthandValue) validate(shortOptions *map[shorthandValue]nameValue, name nameValue) error {
	var errors string

	// Check shorthand does not already exist
	if flagOfShorthand, found := (*shortOptions)[*v]; found {
		errors += extractError(formatErrorItem(2, "-", "shorthand '%s' is already defined for flag '%s'", *v, flagOfShorthand))
	} else if *v != "" {
		(*shortOptions)[*v] = name
	}

	// Check shorthand is one letter only
	if len(*v) > 1 {
		errors += extractError(formatErrorItem(2, "-", "shorthand '%s' is more than one ASCII character", *v))
	}

	if errors != "" {
		return fmt.Errorf("%s", errors)
	}

	return nil
}

// validate checks a usageValue to be defined.
func (v *usageValue) validate() error {
	// Check usage is defined
	if *v == "" {
		return formatErrorItem(2, "-", "no usage defined")
	}

	return nil
}

// validate checks a typeValue and sets a default value if empty.
func (v *typeValue) validate() error {
	// Check type and set default if empty
	if *v == "" {
		*v = FtBool
	}

	// Check type is valid
	implemented, found := ValidTypes[*v]

	if !found {
		return formatErrorItem(2, "-", "type '%s' is not known", *v)
	} else if !implemented {
		return formatErrorItem(2, "-", "type '%s' is not implemented", *v)
	}

	return nil
}

// validate checks a defValueValue and sets a default value if empty.
func (v *defValueValue) validate(aType typeValue) error {
	if *v != "" {
		return nil
	}

	// Init the DefValue
	switch aType {
	case FtBool:
		*v = defValueValue(strconv.FormatBool(false))
	case FtCount, FtDuration, FtFloat32, FtFloat64, FtInt, FtInt8, FtInt16, FtInt32, FtUint, FtUint8, FtUint16, FtUint32:
		*v = defValueValue(strconv.FormatInt(0, 10))
	case FtString:
		*v = ""
	case ftTest3: // used for testing only
		*v = ""
	default:
		if implemented := ValidTypes[aType]; implemented {
			// Mandatory implementation missing -> panic
			panic(fmt.Sprintf("Error implementation of DefValue validation missing for type '%s'", aType))
		}
	}

	return nil
}

// validate checks a noOptDefValValue and sets a default value if needed.
func (v *noOptDefValValue) validate(aType typeValue) error {
	if *v != "" {
		return nil
	}

	// Init the NoOptDefValValue
	switch aType {
	case FtBool:
		*v = noOptDefValValue(strconv.FormatBool(true))
	}

	return nil
}

// validate checks a annotationsValue.
func (v *annotationsValue) validate() error {
	return nil
}

// validateFlag checks all fields by calling the corresponding validate method.
func (f *Flag) validateFlag(longOptions *map[nameValue]bool, shortOptions *map[shorthandValue]nameValue) error {
	errors := ""

	// Chech all fields
	errors += extractError(f.Name.validate(longOptions))
	errors += extractError(f.Shorthand.validate(shortOptions, f.Name))
	errors += extractError(f.Usage.validate())
	errors += extractError(f.Type.validate())
	errors += extractError(f.DefValue.validate(f.Type))
	errors += extractError(f.NoOptDefVal.validate(f.Type))
	errors += extractError(f.Annotations.validate())

	if errors != "" {
		return fmt.Errorf("%s", errors)
	}

	return nil
}

// Init initializes the Flags structure.
func (f *Flags) Init(commandName, script string) {
	f.CommandName = commandName
	f.Script = script
}

// validateFlags checks all Flags by calling the validateFlag method.
func (f *Flags) validateFlags(flags *FlagsDefinition) error {
	var errors string

	// Temporay vars to precheck for duplicated flags. It's still possible
	// other commands will introduce the same flags which is tested
	// afterwards by cobra.
	long := map[nameValue]bool{"help": true}
	short := map[shorthandValue]nameValue{"h": "help"}

	for i := range *flags {
		flag := &(*flags)[i]

		// Additional validations of the flag fields
		flagErrors := extractError(flag.validateFlag(&long, &short))

		if flagErrors != "" {
			errors += extractError(formatErrorItem(1, "*", "for flag '%s':%s", flag.Name, flagErrors))
		}
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

	var err error
	var defs = FlagsDefinition{}

	// Import the JSON to the FlagsDefinition structure and return in case of
	// error
	if err = json.Unmarshal([]byte(data), &defs); err != nil {
		return err
	}

	// Validate the user provided flags and return in case of error
	if err = f.validateFlags(&defs); err != nil {
		return err
	}

	// Assign the data to the field
	f.Definition = defs
	return nil
}

// AssignToCommand iterates the flags and assigns it to the provided command.
func (f *Flags) AssignToCommand(command *cobra.Command) error {
	var errors string

	for _, flag := range f.Definition {
		// Create the flag at the command
		switch flag.Type {
		case FtBool /*, ""*/ : // empty type defaults to bool
			command.Flags().BoolP(string(flag.Name), string(flag.Shorthand), false, string(flag.Usage))
		case FtInt:
			command.Flags().IntP(string(flag.Name), string(flag.Shorthand), 0, string(flag.Usage))
		case FtString:
			command.Flags().StringP(string(flag.Name), string(flag.Shorthand), "", string(flag.Usage))
		case FtUint:
			command.Flags().UintP(string(flag.Name), string(flag.Shorthand), 0, string(flag.Usage))
		default:
			if implemented := ValidTypes[flag.Type]; implemented {
				// Mandatory implementation missing -> panic
				panic(fmt.Sprintf("Error implementation missing for type '%s'", flag.Type))
			}

			continue // continue here, nothing to set for this flag
		}

		// Update default values and annotations
		newFlag := command.Flags().Lookup(string(flag.Name))

		if err := newFlag.Value.Set(string(flag.DefValue)); err != nil {
			// Invalid default value was defined by the user, hide the flag
			// and save the error
			newFlag.Hidden = true
			errors += extractError(formatErrorItem(1, "-", "error '%s' while set value of flag '%s'", err.Error(), flag.Name))
		}

		newFlag.DefValue = string(flag.DefValue)
		newFlag.NoOptDefVal = string(flag.NoOptDefVal)
		newFlag.Annotations = flag.Annotations
	}

	if errors != "" {
		return fmt.Errorf("The following problems were found while assigning the flags to the command '%s' in '%s':%s", f.CommandName, f.Script, errors)
	}

	return nil
}
