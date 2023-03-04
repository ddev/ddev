package util_test

import (
	"github.com/ddev/ddev/pkg/util"
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
	runtime, ok := m["runtime"].(map[string]interface{})
	require.True(t, ok)
	extensions, ok := runtime["extensions"].([]interface{})
	require.True(t, ok)
	assert.Equal("redis", extensions[0])

	//relationships:
	//  database: 'db:mysql'
	relationships, ok := m["relationships"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal("db:mysql", relationships["database"])

	//web:
	//  locations:
	//	  '/':
	//		root: 'web'
	web, ok := m["web"].(map[string]interface{})
	require.True(t, ok)
	locations, ok := web["locations"].(map[string]interface{})
	require.True(t, ok)
	slashloc, ok := locations["/"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal("web", slashloc["root"])

}

func TestYamlToDict(t *testing.T) {
	assert := asrt.New(t)
	f := "testdata/TestYamlToDict/app.yaml"
	m, err := util.YamlFileToMap(f)
	require.NoError(t, err)

	// Start with a top level "base"
	base := make(map[string]interface{})
	// Add dict into it as next layer
	base["dict"], err = util.YamlToDict(m)
	require.NoError(t, err)

	d, ok := base["dict"].(map[string]interface{})
	require.True(t, ok)

	name, ok := d["name"].(string)
	require.True(t, ok)
	assert.Equal("app", name)
}
