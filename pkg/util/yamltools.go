package util

import (
	"bytes"
	"fmt"
	"os"

	"dario.cat/mergo"
	"go.yaml.in/yaml/v4"
)

// YamlFileToMap reads the named file into a map[string]interface{}
func YamlFileToMap(fname string) (map[string]any, error) {
	file, err2 := os.ReadFile(fname)
	contents, err := file, err2
	if err != nil {
		return nil, fmt.Errorf("unable to read file %s (%v)", fname, err)
	}

	itemMap := make(map[string]any)
	err = yaml.Unmarshal(contents, &itemMap)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal %s: %v", contents, err)
	}
	return itemMap, nil
}

// YamlToDict turns random yaml-based interface into a map[string]interface{}
func YamlToDict(topm any) (map[string]any, error) {
	res := make(map[string]any)
	var err error

	switch topm.(type) {
	case map[string]any:
		for yk, v := range topm.(map[string]any) {
			switch v.(type) {
			case string:
				res[yk] = v
			case map[string]any:
				res[yk], err = YamlToDict(v)
			case any:
				res[yk] = v
			default:
				Warning("Configuration has invalid type '%T' for '%s'", v, yk)
				continue
			}
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("yamlToDict: type %T not handled (%v)", topm, topm)
	}
	return res, nil
}

// MergeYamlFiles merges yaml files extraFiles into the baseFile, returning the
// result as a string
// Merging is *override* based, so later files can override contents of others
func MergeYamlFiles(baseFile string, extraFile ...string) (string, error) {
	resultMap, err := YamlFileToMap(baseFile)
	if err != nil {
		return "", err
	}
	for _, fileName := range extraFile {
		m, err := YamlFileToMap(fileName)
		if err != nil {
			return "", err
		}
		err = mergo.Merge(&resultMap, m, mergo.WithOverride)
		if err != nil {
			return "", err
		}
	}
	result, err := yaml.Marshal(resultMap)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// EscapeDollarSign the same thing is done in `docker-compose config`
// See https://github.com/docker/compose/blob/361c0893a9e16d54f535cdb2e764362363d40702/cmd/compose/config.go#L405-L409
func EscapeDollarSign(marshal []byte) []byte {
	dollar := []byte{'$'}
	escDollar := []byte{'$', '$'}
	return bytes.ReplaceAll(marshal, dollar, escDollar)
}
