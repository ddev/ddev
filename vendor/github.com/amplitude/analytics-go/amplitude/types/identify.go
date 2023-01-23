package types

import (
	"fmt"
)

type IdentityOp string

const (
	IdentityOpAdd        IdentityOp = "$add"
	IdentityOpAppend     IdentityOp = "$append"
	IdentityOpClearAll   IdentityOp = "$clearAll"
	IdentityOpPrepend    IdentityOp = "$prepend"
	IdentityOpSet        IdentityOp = "$set"
	IdentityOpSetOnce    IdentityOp = "$setOnce"
	IdentityOpUnset      IdentityOp = "$unset"
	IdentityOpPreInsert  IdentityOp = "$preInsert"
	IdentityOpPostInsert IdentityOp = "$postInsert"
	IdentityOpRemove     IdentityOp = "$remove"

	UnsetValue string = "-"
)

type Identify struct {
	PropertiesSet    map[string]struct{}
	Properties       map[IdentityOp]map[string]interface{}
	validateWarnings []string
}

func (i *Identify) Validate() ([]string, []string) {
	var validateErrors []string
	if len(i.Properties) == 0 {
		validateErrors = append(validateErrors, "Empty Properties")
	}

	return validateErrors, i.validateWarnings
}

func (i *Identify) containsProperty(property string) bool {
	_, ok := i.PropertiesSet[property]

	return ok
}

func (i *Identify) containsOperation(op IdentityOp) bool {
	for operation := range i.Properties {
		if operation == op {
			return true
		}
	}

	return false
}

func (i *Identify) setUserProperty(operation IdentityOp, property string, value interface{}) {
	if len(property) == 0 {
		i.validateWarnings = append(i.validateWarnings, fmt.Sprintf("Attempting to perform operation %s with a null or empty string property, ignoring", string(operation)))
	}

	if value == nil && operation != IdentityOpClearAll {
		i.validateWarnings = append(i.validateWarnings, fmt.Sprintf("Attempting to perform operation %s with null value for property %s, ignoring", string(operation), property))
	}

	if i.containsOperation(IdentityOpClearAll) {
		i.validateWarnings = append(i.validateWarnings, fmt.Sprintf("This Identify already contains a $clearAll operation, ignoring operation %s for property %s", string(operation), property))
	}

	if i.containsProperty(property) {
		i.validateWarnings = append(i.validateWarnings, fmt.Sprintf("Already used property %s in previous operation, ignoring operation %s", property, string(operation)))
	}

	if i.Properties == nil {
		i.Properties = make(map[IdentityOp]map[string]interface{})
	}

	if i.PropertiesSet == nil {
		i.PropertiesSet = make(map[string]struct{})
	}

	if !i.containsOperation(operation) {
		i.Properties[operation] = make(map[string]interface{})
	}

	i.Properties[operation][property] = value
	i.PropertiesSet[property] = struct{}{}
}

// Set sets the value of a user property.
func (i *Identify) Set(property string, value interface{}) *Identify {
	i.setUserProperty(IdentityOpSet, property, value)

	return i
}

// SetOnce sets the value of user property only once.
// Subsequent calls using SetOnce will be ignored.
func (i *Identify) SetOnce(property string, value interface{}) *Identify {
	i.setUserProperty(IdentityOpSetOnce, property, value)

	return i
}

// Add increments a user property by some numerical value.
// If the user property does not have a value set yet,
// it will be initialized to 0 before being incremented.
func (i *Identify) Add(property string, value interface{}) *Identify {
	i.setUserProperty(IdentityOpAdd, property, value)

	return i
}

// Prepend prepends a value or values to a user property array.
// If the user property does not have a value set yet,
// it will be initialized to an empty list before the new values are prepended.
func (i *Identify) Prepend(property string, value interface{}) *Identify {
	i.setUserProperty(IdentityOpPrepend, property, value)

	return i
}

// Append appends a value or values to a user property array.
// If the user property does not have a value set yet,
// it will be initialized to an empty list before the new values are prepended.
func (i *Identify) Append(property string, value interface{}) *Identify {
	i.setUserProperty(IdentityOpAppend, property, value)

	return i
}

// PreInsert pre-inserts a value or values to a user property,
// if it does not exist in the user property yet.
// Pre-insert means inserting the value(s) at the beginning of a given list.
// If the user property does not have a value set yet,
// it will be initialized to an empty list before the new values are pre-inserted.
// If the user property has an existing value, it will be no operation.
func (i *Identify) PreInsert(property string, value interface{}) *Identify {
	i.setUserProperty(IdentityOpPreInsert, property, value)

	return i
}

// PostInsert post-inserts a value or values to a user property,
// if it does not exist in the user property yet.
// Post-insert means inserting the value(s) at the end of a given list.
// If the user property does not have a value set yet,
// it will be initialized to an empty list before the new values are post-inserted.
// If the user property has an existing value, it will be no operation.
func (i *Identify) PostInsert(property string, value interface{}) *Identify {
	i.setUserProperty(IdentityOpPostInsert, property, value)

	return i
}

// Remove removes a value or values to a user property, if it exists in the user property.
// Remove means remove the existing value(s) from the given list.
// If the item does not exist in the user property, it will be no operation.
func (i *Identify) Remove(property string, value interface{}) *Identify {
	i.setUserProperty(IdentityOpRemove, property, value)

	return i
}

// Unset removes the user property from the user profile.
func (i *Identify) Unset(property string) *Identify {
	i.setUserProperty(IdentityOpUnset, property, UnsetValue)

	return i
}

// ClearAll removes all user properties of this user.
func (i *Identify) ClearAll() *Identify {
	i.setUserProperty(IdentityOpClearAll, UnsetValue, nil)

	return i
}
