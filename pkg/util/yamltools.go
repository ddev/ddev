package util

import (
	"fmt"
	"os"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"
)

// YamlFileToMap() reads the named file into a map[string]interface{}
func YamlFileToMap(fname string) (map[string]interface{}, error) {
	file, err2 := os.ReadFile(fname)
	contents, err := file, err2
	if err != nil {
		return nil, fmt.Errorf("unable to read file %s (%v)", fname, err)
	}

	itemMap := make(map[string]interface{})
	err = yaml.Unmarshal(contents, &itemMap)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal %s: %v", contents, err)
	}
	return itemMap, nil
}

// YamlToDict turns random yaml-based interface into a map[string]interface{}
func YamlToDict(topm interface{}) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	var err error

	switch topm.(type) {
	case map[string]interface{}:
		for yk, v := range topm.(map[string]interface{}) {
			switch v.(type) {
			case string:
				res[yk] = v
			case map[string]interface{}:
				res[yk], err = YamlToDict(v)
			case interface{}:
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
