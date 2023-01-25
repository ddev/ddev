package util

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
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
	case map[interface{}]interface{}:
		for yk, v := range topm.(map[interface{}]interface{}) {
			ys, ok := yk.(string)
			// TODO: Explain the problem, figure out if this is a golang/yaml bug
			if !ok {
				Warning("%v can't be converted to string")
				continue
			}
			switch v.(type) {
			case string:
				res[ys] = v
			case map[interface{}]interface{}:
				res[ys], err = YamlToDict(v)
			case map[string]interface{}:
				res[ys], err = YamlToDict(v)
			case []interface{}:
				res[ys] = v
			case interface{}:
				res[ys] = v
			default:
				return nil, fmt.Errorf("YamlToDict: type %T not handled (%v)", yk, yk)
			}
			if err != nil {
				return nil, err
			}
		}
	case map[string]interface{}:
		for yk, v := range topm.(map[string]interface{}) {
			switch v.(type) {
			case string:
				res[yk] = v
			case map[interface{}]interface{}:
				res[yk], err = YamlToDict(v)
			case map[string]interface{}:
				res[yk], err = YamlToDict(v)
			case interface{}:
				res[yk] = v
			default:
				return nil, fmt.Errorf("YamlToDict: type %T not handled (%v)", yk, yk)
			}
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("YamlToDict: type %T not handled (%v)", topm, topm)
	}
	return res, nil
}
