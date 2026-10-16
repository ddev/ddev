package util

import (
	"os"
	"regexp"
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

var envAssignmentStartRegex = regexp.MustCompile(`^[ \t]*[A-Za-z_][A-Za-z0-9_]*=`)

// EnvFlagEntries returns the NAME=value and bare NAME entries from every use
// of a repeatable environment flag, --web-environment or --environment.
//
// A comma starts a new entry when NAME= follows it or when it ends a bare
// NAME. Any other comma belongs to the value, so `TOKEN=a,b,ENV=main` is two
// entries and `METHODS=GET,POST` one. An empty or quoted empty flag value
// adds nothing.
func EnvFlagEntries(flagValues []string) []string {
	entries := []string{}
	for _, value := range flagValues {
		value = strings.TrimSpace(value)
		if value == "" || value == `""` || value == `''` {
			continue
		}
		start := len(entries)
		for part := range strings.SplitSeq(value, ",") {
			newEntry := len(entries) == start ||
				envAssignmentStartRegex.MatchString(part) ||
				!strings.Contains(entries[len(entries)-1], "=")
			switch {
			case !newEntry:
				entries[len(entries)-1] += "," + part
			case strings.TrimSpace(part) != "":
				entries = append(entries, strings.TrimLeft(part, " \t"))
			}
		}
	}
	return entries
}
