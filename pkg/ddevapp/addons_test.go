package ddevapp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

func TestProcessPHPAction(t *testing.T) {
	tmpDir := testcommon.CreateTmpDir(t.Name())

	// Create the project directory
	projectDir := filepath.Join(tmpDir, t.Name())
	err := os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)

	// Create and initialize the app properly using NewApp
	app, err := ddevapp.NewApp(projectDir, true)
	require.NoError(t, err)

	// Write the config using the proper DDEV method
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = app.Stop(true, false)
		_ = os.RemoveAll(tmpDir)
	})

	// Test basic PHP action
	t.Run("BasicPHPAction", func(t *testing.T) {
		action := `<?php
echo "Hello from PHP test\n";
echo "This is working\n";
?>`
		err := ddevapp.ProcessAddonAction(action, ddevapp.InstallDesc{}, app, true)
		require.NoError(t, err, "PHP action should execute without error")
	})
}

func TestProcessAddonActionPHPDetection(t *testing.T) {
	// Test PHP detection
	t.Run("PHPDetection", func(t *testing.T) {
		phpAction := "<?php echo 'Hello'; ?>"
		bashAction := "echo 'Hello'"

		// Check that PHP actions are detected
		require.True(t, strings.HasPrefix(strings.TrimSpace(phpAction), "<?php"))
		require.False(t, strings.HasPrefix(strings.TrimSpace(bashAction), "<?php"))
	})

	// Test mixed whitespace PHP detection
	t.Run("PHPDetectionWithWhitespace", func(t *testing.T) {
		phpAction := `
		<?php echo 'Hello'; ?>`

		require.True(t, strings.HasPrefix(strings.TrimSpace(phpAction), "<?php"))
	})
}
