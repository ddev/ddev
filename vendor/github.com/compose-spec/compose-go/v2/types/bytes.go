/*
   Copyright 2020 The Compose Specification Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package types

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
)

// UnitBytes is the bytes type
type UnitBytes int64

// MarshalYAML makes UnitBytes implement yaml.Marshaller
func (u UnitBytes) MarshalYAML() (interface{}, error) {
	return fmt.Sprintf("%d", u), nil
}

// MarshalJSON makes UnitBytes implement json.Marshaler
func (u UnitBytes) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%d"`, u)), nil
}

// byteValue is the shape of a byte value as a string: an amount, which may be
// fractional or use an exponent, optionally followed by a unit. It leaves out
// the notations that strconv.ParseFloat would accept but that make no sense
// for a size (hexadecimal floats, "inf", "nan", digit separators, signs).
var byteValue = regexp.MustCompile(`^(\d+\.?\d*|\.\d+)([eE][+-]?\d+)? ?([A-Za-z]*)$`)

// maxBytes is 2^63 as a float64: the smallest amount of bytes that does not fit
// in a UnitBytes.
const maxBytes = float64(1 << 63)

// parseString parses a string into a UnitBytes value, supporting plain
// integers, negative values (e.g., "-1"), and human-readable byte units
// (e.g., "1g", "512m").
//
// An amount that does not fit in 64 bits is rejected, as is a fractional
// number of bytes (e.g., "0.5"): both used to be turned silently into a
// different value. A fraction of a larger unit (e.g., "1.5g") is accepted and
// rounded down to a whole number of bytes.
func (u *UnitBytes) parseString(s string) error {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		*u = UnitBytes(n)
		return nil
	}
	m := byteValue.FindStringSubmatch(s)
	if m == nil {
		return fmt.Errorf("invalid size: '%s'", s)
	}
	multiplier, ok := unitMultiplier(m[3])
	if !ok {
		return fmt.Errorf("invalid suffix: '%s'", strings.ToLower(m[3]))
	}
	amount, err := strconv.ParseFloat(m[1]+m[2], 64)
	if err != nil {
		return fmt.Errorf("invalid size: '%s': %w", s, err)
	}
	return u.setBytes(s, amount*multiplier, multiplier == 1)
}

// unitMultiplier returns the number of bytes in a unit: b, or k, m, g, t and p
// (powers of 1024), each optionally followed by b or ib, in any case. No unit
// stands for bytes.
func unitMultiplier(unit string) (float64, bool) {
	unit = strings.ToLower(unit)
	if unit == "" || unit == "b" {
		return 1, true
	}
	power := strings.IndexByte("kmgtp", unit[0]) + 1
	if power == 0 {
		return 0, false
	}
	if rest := unit[1:]; rest != "" && rest != "b" && rest != "ib" {
		return 0, false
	}
	return math.Pow(1024, float64(power)), true
}

// setBytes sets u to a computed number of bytes, rejecting a value that does not
// fit. In bytes, the amount must be a whole number as well.
func (u *UnitBytes) setBytes(original string, bytes float64, inBytes bool) error {
	switch {
	case bytes >= maxBytes || bytes < -maxBytes:
		return fmt.Errorf("invalid size: '%s': out of range, must fit in a signed 64 bits integer", original)
	case inBytes && bytes != math.Trunc(bytes):
		return fmt.Errorf("invalid size: '%s': a number of bytes must be an integer", original)
	}
	*u = UnitBytes(bytes)
	return nil
}

// UnmarshalJSON makes UnitBytes implement json.Unmarshaler
func (u *UnitBytes) UnmarshalJSON(data []byte) error {
	var v int64
	if err := json.Unmarshal(data, &v); err == nil {
		*u = UnitBytes(v)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		return u.parseString(s)
	}
	// a number that is not an integer literal, such as 1e3 or 1.5
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	return u.parseString(n.String())
}

// UnmarshalYAML makes UnitBytes implement yaml.Unmarshaler
func (u *UnitBytes) UnmarshalYAML(value *yaml.Node) error {
	var v int64
	if err := value.Decode(&v); err == nil {
		*u = UnitBytes(v)
		return nil
	}
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	return u.parseString(s)
}

// DecodeMapstructure decodes a byte value from the YAML model: a number of
// bytes, or a string with an optional unit. A number that is not a whole
// number of bytes, or does not fit, is an error rather than a silent 0.
func (u *UnitBytes) DecodeMapstructure(value interface{}) error {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		return u.parseString(v)
	}
	// the YAML decoder produces int, int64, uint64 and float64, a model built by
	// hand may hold any numeric type, UnitBytes included
	switch v := reflect.ValueOf(value); v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		*u = UnitBytes(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if v.Uint() > math.MaxInt64 {
			return fmt.Errorf("invalid size: '%d': out of range, must fit in a signed 64 bits integer", v.Uint())
		}
		*u = UnitBytes(v.Uint())
	case reflect.Float32, reflect.Float64:
		return u.setBytes(strconv.FormatFloat(v.Float(), 'g', -1, 64), v.Float(), true)
	default:
		return fmt.Errorf("unexpected value type %T for a byte value", value)
	}
	return nil
}
