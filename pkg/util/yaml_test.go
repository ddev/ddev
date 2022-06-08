package util_test

import (
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestYamlFileToMap tests YamlFileToMap()
func TestYamlFileToMap(t *testing.T) {
	assert := asrt.New(t)
	f := "testdata/TestYamlFileToMap/app.yaml"
	m, err := util.YamlFileToMap(f)
	require.NoError(t, err)

	// Check app: name
	assert.Equal("app", m["name"])

	//runtime:
	//  extensions:
	//	  - redis
	runtime, ok := m["runtime"].(map[interface{}]interface{})
	require.True(t, ok)
	extensions, ok := runtime["extensions"].([]interface{})
	require.True(t, ok)
	assert.Equal("redis", extensions[0])

	//relationships:
	//  database: 'db:mysql'
	relationships, ok := m["relationships"].(map[interface{}]interface{})
	require.True(t, ok)
	assert.Equal("db:mysql", relationships["database"])

	//web:
	//  locations:
	//	  '/':
	//		root: 'web'
	web, ok := m["web"].(map[interface{}]interface{})
	require.True(t, ok)
	locations, ok := web["locations"].(map[interface{}]interface{})
	require.True(t, ok)
	slashloc, ok := locations["/"].(map[interface{}]interface{})
	require.True(t, ok)
	assert.Equal("web", slashloc["root"])

}
