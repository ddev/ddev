package ddevapp

import (
	"strings"
)

// EnvToUniqueEnv() makes sure that only the last occurrence of an env (NAME=val or bare NAME)
// slice is actually retained. Bare variable names without a value (e.g. "MY_VAR") are passed
// through as-is; docker-compose resolves them from the host environment at container start time.
func EnvToUniqueEnv(inSlice *[]string) []string {
	mapStore := map[string]string{}

	for _, s := range *inSlice {
		// Both "KEY=value" and bare "KEY" are supported.
		// strings.Cut returns the part before "=" as the key in both cases.
		// Last entry for a given key wins.
		k, _, _ := strings.Cut(s, "=")
		mapStore[k] = s
	}
	newSlice := make([]string, 0, len(mapStore))
	for _, v := range mapStore {
		newSlice = append(newSlice, v)
	}
	if len(newSlice) == 0 {
		return nil
	}
	return newSlice
}
