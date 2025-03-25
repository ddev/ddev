package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

// TestMergeYamlFiles tests many permutations of MergetYamlFiles()
func TestMergeYamlFiles(t *testing.T) {
	origDir, _ := os.Getwd()
	testData := filepath.Join(origDir, "testdata", t.Name())

	testCases := []struct {
		content string
		dir     string
	}{
		{"Basic no-actual-merge, just using the base file", "baseFileOnly"},
		{"Basic file with a single file that only adds, no merging", "baseWithPlugins"},
		{"certificates resolvers", "baseWithCertificatesResolvers"},
		{"Overrides", "Overrides"},
		{"caServerOverride", "caServerOverride"},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			testDir := filepath.Join(testData, tc.dir)
			extraFiles, err := filepath.Glob(filepath.Join(testDir, "extra*.yaml"))
			require.NoError(t, err)

			mergeResult, err := util.MergeYamlFiles(filepath.Join(testDir, "base.yaml"), extraFiles...)
			require.NoError(t, err)

			expectedResultFile := filepath.Join(testDir, "expectation.yaml")
			expectedResultString, err := fileutil.ReadFileIntoString(expectedResultFile)
			require.NoError(t, err)
			// Unmarshall the loaded result expectation so it will look the same as merged (without comments, etc)
			var tmpMap map[string]interface{}
			err = yaml.Unmarshal([]byte(expectedResultString), &tmpMap)
			require.NoError(t, err)
			unmarshalledExpectationString, err := yaml.Marshal(tmpMap)
			require.NoError(t, err)

			require.Equal(t, string(unmarshalledExpectationString), mergeResult)
		})
	}
}
