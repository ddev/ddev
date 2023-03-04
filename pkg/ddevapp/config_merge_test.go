package ddevapp_test

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestConfigMerge takes a variety of test expectations and checks to see if they work out
func TestConfigMerge(t *testing.T) {
	assert := asrt.New(t)

	// Each item here has a set of config.*.yaml to be composed, and a config.yaml that should reflect the result
	for _, c := range []string{"envtest", "fulloverridetest", "hookstest", "overridetest", "scalartest"} {
		composedApp := SetupTestTempDir(t, filepath.Join(c, "components"))
		expectedApp := SetupTestTempDir(t, filepath.Join(c, "expected"))
		composedApp.Name = ""
		composedApp.AppRoot = ""
		composedApp.ConfigPath = ""
		expectedApp.Name = ""
		expectedApp.AppRoot = ""
		expectedApp.ConfigPath = ""
		// We don't know in advance the ordering of the WebEnvironment for testing purposes,
		// and it doesn't matter in reality, so sort both for testing before comparison
		sort.Strings(expectedApp.WebEnvironment)
		sort.Strings(composedApp.WebEnvironment)

		assert.Equal(expectedApp, composedApp, "%s failed", c)
	}
}

// TestConfigMergeStringList verifies that scalar string merges update w/o clobber
func TestConfigMergeStringList(t *testing.T) {
	assert := asrt.New(t)
	app := SetupTestTempDir(t, "")

	// test matches allowing for delete syntax (a prefixed !)
	assertSimpleMatch := func(expected bool, match string, setting []string) {
		test := assert.True
		if !expected {
			test = assert.False
		}

		for _, val := range setting {
			if val == match {
				test(true, match)
				return
			}
		}
		test(false, match)
	}
	// no clobber of old setting
	assertSimpleMatch(true, "somename", app.AdditionalHostnames)
	// successful merge
	assertSimpleMatch(true, "somename-new", app.AdditionalHostnames)
}

// TestConfigMergeEnvItems verifies that config overrides web_environment
// and do not destroy (but may update) config.yaml stuff
func TestConfigMergeEnvItems(t *testing.T) {
	assert := asrt.New(t)
	app := SetupTestTempDir(t, "")

	// check the config file w/o overrides
	noOverridesApp, err := ddevapp.NewApp(app.AppRoot, false)
	require.NoError(t, err)
	assert.IsType([]string{}, noOverridesApp.WebEnvironment)

	// The app loaded without overrides should get the original values expected here
	for _, v := range []string{`LARRY=l`, `MOE=m`, `CURLEY=c`} {
		assert.Contains(noOverridesApp.WebEnvironment, v, "the app without overrides should have had %v but it didn't, webEnvironment=%v", v, noOverridesApp.WebEnvironment)
	}

	// With overrides we should have different values with the config.override.yaml values added
	withOverridesApp, err := ddevapp.NewApp(app.AppRoot, true)
	require.NoError(t, err)

	for _, v := range []string{`LARRY=lz`, `MOE=mz`, `CURLEY=c`, `SHEMP=s`} {
		assert.Contains(withOverridesApp.WebEnvironment, v, "the app without overrides should have had %v but it didn't, webEnvironment=%v", v, noOverridesApp.WebEnvironment)
	}

}

// TestConfigHooksMerge makes sure that hooks get merged with additional config.*.yaml
func TestConfigHooksMerge(t *testing.T) {
	app := SetupTestTempDir(t, "")

	// some helpers
	getHookTasks := func(hooks map[string][]ddevapp.YAMLTask, hook string) []ddevapp.YAMLTask {
		tasks, ok := hooks[hook]
		if !ok {
			return nil
		}
		return tasks
	}

	// the fields we want are private!
	hasTask := func(hooks map[string][]ddevapp.YAMLTask, hook, taskKey, desc string) bool {
		tasks := getHookTasks(hooks, hook)
		if tasks == nil {
			return false
		}
		found := false
		for _, task := range tasks {
			taskDesc := ""
			taskInterface, ok := task[taskKey]
			if !ok {
				// we guessed the key wrong
				continue
			}

			taskDesc, ok = taskInterface.(string)
			if !ok {
				// we expected the command to be a string, but WTF?
				continue
			}

			//t.Logf("key %s as %s", taskKey, taskDesc)

			if taskDesc == desc {
				found = true
			}

		}
		return found
	}

	assertTask := func(expected bool, hook, taskKey, desc string) {
		tasks := getHookTasks(app.Hooks, hook)
		if tasks == nil {
			if expected {
				t.Errorf("did not found tasks for %s", hook)
			} else {
				return
			}
		} else {
			found := hasTask(app.Hooks, hook, taskKey, desc)
			if found != expected {
				msg := "Expected "
				if !expected {
					msg = "Did not expect "
				}
				t.Errorf("%s Hook %s with %s", msg, hook, desc)
			}
		}
	}

	assertTask(true, "post-start", "exec", "simple random expression")
	assertTask(true, "post-start", "exec-host", "simple host command")
	assertTask(true, "post-start", "exec", "simple web command")
	assertTask(true, "post-import-db", "exec", "drush uli")

}

// SetupTestTempDir creates the test directory and related objects.
func SetupTestTempDir(t *testing.T, subDir string) *ddevapp.DdevApp {
	assert := asrt.New(t)

	projDir, err := filepath.Abs(testcommon.CreateTmpDir(t.Name()))
	require.NoError(t, err)

	testConfig := filepath.Join("./testdata/", t.Name(), subDir, "/.ddev")
	err = fileutil.CopyDir(testConfig, filepath.Join(projDir, ".ddev"))
	require.NoError(t, err)

	app, err := ddevapp.NewApp(projDir, true)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.RemoveAll(projDir)
		assert.NoError(err)
	})

	return app
}

// TestEnvToUniqueEnv tests EnvToUniqueEnv
func TestEnvToUniqueEnv(t *testing.T) {
	assert := asrt.New(t)

	testBedSources := [][]string{
		{"ONE=one", "ONE=two", "ONE=three", "TWO=two", "TWO=three", "TWO=four"},
	}

	testBedExpectations := [][]string{
		{"ONE=three", "TWO=four"},
	}

	for i := 0; i < len(testBedSources); i++ {
		res := ddevapp.EnvToUniqueEnv(&testBedSources[i])
		sort.Strings(res)
		assert.Equal(testBedExpectations[i], res)
	}
}
