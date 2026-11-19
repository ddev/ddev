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

package validation

import (
	"fmt"
	"math"
	"net"
	"reflect"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/tree"
)

type checkerFunc func(value any, p tree.Path) error

var checks = map[tree.Path]checkerFunc{
	"volumes.*":                                          checkVolume,
	"configs.*":                                          checkFileObject("file", "environment", "content"),
	"secrets.*":                                          checkFileObject("file", "environment"),
	"services.*.ports.*":                                 checkIPAddress,
	"services.*.mem_swappiness":                          checkMemSwappiness,
	"services.*.pre_start.*.mem_swappiness":              checkMemSwappiness,
	"jobs.*.mem_swappiness":                              checkMemSwappiness,
	"services.*.mem_limit":                               checkNotNegative,
	"jobs.*.mem_limit":                                   checkNotNegative,
	"services.*.mem_reservation":                         checkNotNegative,
	"jobs.*.mem_reservation":                             checkNotNegative,
	"services.*.shm_size":                                checkNotNegative,
	"jobs.*.shm_size":                                    checkNotNegative,
	"services.*.build.shm_size":                          checkNotNegative,
	"jobs.*.build.shm_size":                              checkNotNegative,
	"services.*.pre_start.*.mem_limit":                   checkNotNegative,
	"services.*.pre_start.*.mem_reservation":             checkNotNegative,
	"services.*.pre_start.*.shm_size":                    checkNotNegative,
	"services.*.deploy.resources.limits.memory":          checkNotNegative,
	"services.*.deploy.resources.reservations.memory":    checkNotNegative,
	"services.*.develop.watch.*.path":                    checkPath,
	"services.*.deploy.resources.reservations.devices.*": checkDeviceRequest,
	"services.*.gpus.*":                                  checkDeviceRequest,
}

func Validate(dict map[string]any) error {
	return check(dict, tree.NewPath())
}

func check(value any, p tree.Path) error {
	for pattern, fn := range checks {
		if p.Matches(pattern) {
			return fn(value, p)
		}
	}
	switch v := value.(type) {
	case map[string]any:
		for k, v := range v {
			err := check(v, p.Next(k))
			if err != nil {
				return err
			}
		}
	case []any:
		for _, e := range v {
			err := check(e, p.Next("[]"))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func checkFileObject(keys ...string) checkerFunc {
	return func(value any, p tree.Path) error {
		v := value.(map[string]any)
		count := 0
		for _, s := range keys {
			if _, ok := v[s]; ok {
				count++
			}
		}
		if count > 1 {
			return fmt.Errorf("%s: %s attributes are mutually exclusive", p, strings.Join(keys, "|"))
		}
		if count == 0 {
			if _, ok := v["driver"]; ok {
				// User specified a custom driver, which might have it's own way to set content
				return nil
			}
			if _, ok := v["external"]; !ok {
				return fmt.Errorf("%s: one of %s must be set", p, strings.Join(keys, "|"))
			}
		}
		return nil
	}
}

func checkPath(value any, p tree.Path) error {
	v := value.(string)
	if v == "" {
		return fmt.Errorf("%s: value can't be blank", p)
	}
	return nil
}

func checkDeviceRequest(value any, p tree.Path) error {
	v := value.(map[string]any)
	_, hasCount := v["count"]
	_, hasIDs := v["device_ids"]
	if hasCount && hasIDs {
		return fmt.Errorf(`%s: "count" and "device_ids" attributes are exclusive`, p)
	}
	return nil
}

func checkIPAddress(value any, p tree.Path) error {
	if v, ok := value.(map[string]any); ok {
		ip, ok := v["host_ip"]
		if ok && net.ParseIP(ip.(string)) == nil {
			return fmt.Errorf("%s: invalid ip address: %s", p, ip)
		}
	}
	return nil
}

// checkMemSwappiness checks mem_swappiness is a percentage, as the specification
// requires. The attribute is modeled as a UnitBytes, which would otherwise accept
// any amount, in any byte unit.
//
// -1 is accepted although the specification only documents 0 to 100: it is the
// default of the Docker CLI (--memory-swappiness), which the engine reads as
// "not set", and files that spell it out have always been accepted.
func checkMemSwappiness(value any, p tree.Path) error {
	var swappiness int64
	// the model may come from a pre-parsed map, with numbers of any Go type
	switch v := reflect.ValueOf(value); v.Kind() {
	case reflect.Invalid:
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		swappiness = v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if v.Uint() > math.MaxInt64 {
			return invalidMemSwappiness(p, value)
		}
		swappiness = int64(v.Uint())
	case reflect.Float32, reflect.Float64:
		// the schema types the attribute as an integer, which 1e2 is
		f := v.Float()
		if f != math.Trunc(f) || f < -1 || f > 100 {
			return invalidMemSwappiness(p, value)
		}
		swappiness = int64(f)
	case reflect.String:
		n, err := strconv.ParseInt(v.String(), 10, 64)
		if err != nil {
			return invalidMemSwappiness(p, value)
		}
		swappiness = n
	default:
		return invalidMemSwappiness(p, value)
	}
	if swappiness < -1 || swappiness > 100 {
		return invalidMemSwappiness(p, value)
	}
	return nil
}

// checkNotNegative checks a byte value is not negative. Such attributes are
// modeled as a UnitBytes, which accepts -1 because memswap_limit needs it for
// unlimited swap. For every other amount (memory limits and reservations, shared
// memory size) the Docker CLI rejects a negative value, so does Compose.
func checkNotNegative(value any, p tree.Path) error {
	negative := false
	// the model may come from a pre-parsed map, with numbers of any Go type
	switch v := reflect.ValueOf(value); v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		negative = v.Int() < 0
	case reflect.Float32, reflect.Float64:
		negative = v.Float() < 0
	case reflect.String:
		// anything else than an integer is for the byte value parser to judge
		if n, err := strconv.ParseInt(v.String(), 10, 64); err == nil {
			negative = n < 0
		}
	}
	if negative {
		return fmt.Errorf("%s: invalid value %v: must not be negative", p, value)
	}
	return nil
}

func invalidMemSwappiness(p tree.Path, value any) error {
	return fmt.Errorf("%s: invalid value %v: mem_swappiness must be a percentage, an integer between 0 and 100", p, value)
}
