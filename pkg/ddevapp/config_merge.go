package ddevapp

import (
	"fmt"
	"strings"
)

// EnvToUniqueEnv() makes sure that only the last occurrence of an env (NAME=val)
// slice is actually retained.
func EnvToUniqueEnv(inSlice *[]string) []string {
	mapStore := map[string]string{}
	newSlice := []string{}

	for _, s := range *inSlice {
		// config.yaml vars look like ENV1=val1 and ENV2=val2
		// Split them and then make sure the last one wins
		k, v, found := strings.Cut(s, "=")
		// If we didn't find the "=" delimiter, it wasn't an env
		if !found {
			continue
		}
		mapStore[k] = v
	}
	for k, v := range mapStore {
		newSlice = append(newSlice, fmt.Sprintf("%s=%v", k, v))
	}
	if len(newSlice) == 0 {
		return nil
	}
	return newSlice
}
