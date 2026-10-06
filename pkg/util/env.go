package util

import (
	"os"
	"strings"
)

// HostEnv returns NAME=value for each of names that is set on the host. An
// empty value is kept, because libraries that only check whether a variable
// exists should see the same thing in the container as on the host.
func HostEnv(names ...string) []string {
	var env []string
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return env
}

// MergeEnv joins NAME=value (or bare NAME) lists into one, keeping only the
// last entry for each name, so a later list overrides an earlier one. Each
// entry keeps its position, which makes the result stable.
func MergeEnv(lists ...[]string) []string {
	var all []string
	for _, list := range lists {
		all = append(all, list...)
	}
	last := make(map[string]int, len(all))
	for i, entry := range all {
		name, _, _ := strings.Cut(entry, "=")
		last[name] = i
	}
	var merged []string
	for i, entry := range all {
		name, _, _ := strings.Cut(entry, "=")
		if last[name] == i {
			merged = append(merged, entry)
		}
	}
	return merged
}
