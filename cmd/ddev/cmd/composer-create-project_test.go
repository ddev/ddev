package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/composer"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

func TestComposerCreateProjectCmd(t *testing.T) {
	testcommon.SkipIfGotestShort(t)
	composerVersionForThisTest := nodeps.ComposerDefault
	//composerVersionForThisTest := "2.8.0"

	origDir, err := os.Getwd()
	require.NoError(t, err)

	// The commands tested below differ only by docroot and by php versus any
	// other type. The quickstart tests run create-project for each project type.
	for _, docRoot := range []string{"", "doc-root"} {
		for _, projectType := range []string{nodeps.AppTypePHP, nodeps.AppTypeDrupal} {
			t.Logf("== BEGIN TestComposerCreateProjectCmd for project of type '%s' with docroot '%s'\n", projectType, docRoot)
			tmpDir := testcommon.CreateTmpDir(t.Name() + projectType)
			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Prepare arguments
			arguments := []string{"config", "--project-type", projectType, "--composer-version", composerVersionForThisTest}

			composerDirOnHost := tmpDir
			if docRoot != "" {
				arguments = append(arguments, "--docroot", docRoot)
				// For Drupal, we arbitrarily place the composer root to be the docroot
				// Normally for Drupal the docroot would be web, and the composer root would be the
				// project root (default). But here we're making sure we can use the docroot
				// as the composer_root. Acquia sites often do this...
				if projectType == nodeps.AppTypeDrupal {
					arguments = append(arguments, "--composer-root", docRoot)
					composerDirOnHost = filepath.Join(tmpDir, docRoot)
				}
			}

			// Basic config
			_, err = exec.RunHostCommand(DdevBin, arguments...)
			require.NoError(t, err)

			// Test trivial command
			out, err := exec.RunHostCommand(DdevBin, "composer", "--version")
			require.NoError(t, err)
			require.Contains(t, out, "Composer version")

			// Get an app so we can do waits
			app, err := ddevapp.NewApp(tmpDir, true)
			require.NoError(t, err)

			t.Cleanup(func() {
				err = app.Stop(true, false)
				require.NoError(t, err)
				err = os.Chdir(origDir)
				require.NoError(t, err)
				_ = os.RemoveAll(tmpDir)
			})

			err = app.Start()
			require.NoError(t, err)

			out, err = exec.RunHostCommand(DdevBin, "composer", "--version")
			require.NoError(t, err)
			require.Contains(t, out, fmt.Sprintf("Composer version %s", composerVersionForThisTest))

			cmd := ""
			// These are different conditions to test different composer flag combinations
			// Conditions for docRoot and projectType are not important here, they are only needed to make the test act different each time
			if docRoot == "" {
				cmd = "ddev composer create-project --no-plugins --no-scripts ddev/ddev-test-composer-create"
				if projectType == nodeps.AppTypePHP {
					cmd = "ddev composer create-project --no-install ddev/ddev-test-composer-create ."
				}
			} else {
				if projectType != nodeps.AppTypePHP {
					cmd = "ddev composer create-project ddev/ddev-test-composer-create --prefer-install auto . --no-dev v1.0.0"
				} else {
					cmd = "ddev composer create-project -vvv ddev/ddev-test-composer-create --prefer-install=auto --fake-flag ."
				}
			}

			t.Logf("Attempting cmd='%s' with docroot='%s' composer_root='%s' type='%s'", cmd, docRoot, docRoot, projectType)
			args := strings.Split(strings.TrimPrefix(cmd, "ddev "), " ")

			// If a file exists in the composer root then composer create-project should fail
			file, err := os.Create(filepath.Join(composerDirOnHost, "touch1.txt"))
			require.NoError(t, err)
			out, err = exec.RunHostCommand(DdevBin, args...)
			require.Error(t, err)
			require.Contains(t, out, "touch1.txt")
			_ = file.Close()
			_ = os.Remove(filepath.Join(composerDirOnHost, "touch1.txt"))

			// Test success
			out, err = exec.RunHostCommand(DdevBin, args...)
			require.NoError(t, err, "['%s'] failed to run %v: err=%v, output=\n=====\n%s\n=====\n", cmd, args, err, out)
			require.Contains(t, out, "Created project in ")
			require.FileExists(t, filepath.Join(composerDirOnHost, "composer.json"))

			if cmd == "ddev composer create-project --no-plugins --no-scripts ddev/ddev-test-composer-create" {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project --no-plugins --no-scripts --no-install ddev/ddev-test-composer-create /tmp/")
				require.NotContains(t, out, "Executing Composer command: [composer run-script post-root-package-install")
				require.Contains(t, out, "Executing Composer command: [composer install --no-plugins --no-scripts]")
				require.NotContains(t, out, "Executing Composer command: [composer run-script post-create-project-cmd")
				// Check the actual result of executing composer scripts
				require.NoFileExists(t, filepath.Join(composerDirOnHost, "created-by-post-root-package-install"))
				require.NoFileExists(t, filepath.Join(composerDirOnHost, "created-by-post-create-project-cmd"))
				// Check vendor directory
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "autoload.php"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require", "composer.json"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require-dev", "composer.json"))
			}

			if cmd == "ddev composer create-project -vvv ddev/ddev-test-composer-create --prefer-install=auto --fake-flag ." {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project -vvv --prefer-install=auto --no-plugins --no-scripts --no-install ddev/ddev-test-composer-create /tmp/")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-root-package-install -vvv]")
				require.Contains(t, out, "Executing Composer command: [composer install -vvv --prefer-install=auto]")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-create-project-cmd -vvv]")
				require.NotContains(t, out, "--fake-flag")
				// Check the actual result of executing composer scripts
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-root-package-install"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-create-project-cmd"))
				// Check vendor directory
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "autoload.php"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require", "composer.json"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require-dev", "composer.json"))
			}

			if cmd == "ddev composer create-project --no-install ddev/ddev-test-composer-create ." {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project --no-install --no-plugins --no-scripts ddev/ddev-test-composer-create /tmp/")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-root-package-install]")
				require.NotContains(t, out, "Executing Composer command: [composer install")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-create-project-cmd]")
				// Check the actual result of executing composer scripts
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-root-package-install"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-create-project-cmd"))
				// Check vendor directory
				require.NoDirExists(t, filepath.Join(composerDirOnHost, "vendor"))
			}

			if cmd == "ddev composer create-project ddev/ddev-test-composer-create --prefer-install auto . --no-dev v1.0.0" {
				// Check what was executed or not
				require.Contains(t, out, "Executing Composer command: [composer create-project --prefer-install auto --no-dev --no-plugins --no-scripts --no-install ddev/ddev-test-composer-create /tmp/")
				require.Contains(t, out, "v1.0.0")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-root-package-install --no-dev]")
				require.Contains(t, out, "Executing Composer command: [composer install --prefer-install auto --no-dev]")
				require.Contains(t, out, "Executing Composer command: [composer run-script post-create-project-cmd --no-dev]")
				// Check the actual result of executing composer scripts
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-root-package-install"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "created-by-post-create-project-cmd"))
				// Check vendor directory
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "autoload.php"))
				require.FileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require", "composer.json"))
				require.NoFileExists(t, filepath.Join(composerDirOnHost, "vendor", "ddev", "ddev-test-composer-require-dev", "composer.json"))
			}

			require.Contains(t, out, "Moving install to Composer root")
			require.Contains(t, out, "ddev composer create-project was successful")

			// Check that resulting composer.json (copied from testdata) has post-root-package-install and post-create-project-cmd scripts
			composerManifest, err := composer.NewManifest(filepath.Join(composerDirOnHost, "composer.json"))
			require.NoError(t, err)
			require.True(t, composerManifest != nil)
			require.True(t, composerManifest.HasPostRootPackageInstallScript())
			require.True(t, composerManifest.HasPostCreateProjectCmdScript())

			err = app.Stop(true, false)
			require.NoError(t, err)
		}
	}
}

func TestComposerCreateAutocomplete(t *testing.T) {
	// DDEV_DEBUG may result in extra output that we don't want
	t.Setenv("DDEV_DEBUG", "")
	// Change to the directory for the project to test.
	// We don't really care what the project is, they should
	// all have composer installed in the web container.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(TestSites[0].Dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		require.NoError(t, err)
	})

	// Make sure the sites exist and are running
	err = addSites()
	require.NoError(t, err)

	// Pressing tab after `composer completion` should result in the completion "bash"
	out, err := exec.RunHostCommand(DdevBin, "__complete", "composer", "create-project", "--")
	require.NoError(t, err)
	// Completions are terminated with ":4", so just grab the stuff before that
	completions, _, found := strings.Cut(out, ":")
	require.True(t, found)
	// We don't need to check all of the possible options - just check that
	// we're getting some completion suggestions that make sense and not just garbage
	require.Contains(t, completions, "--no-install")
	require.Contains(t, completions, "--no-scripts")
	require.Contains(t, completions, "--keep-vcs")
}

func TestComposerCreateProjectPipedInput(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := testcommon.CreateTmpDir(t.Name())
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	_, err = exec.RunHostCommand(DdevBin, "config", "--project-type", nodeps.AppTypePHP)
	require.NoError(t, err)
	app, err := ddevapp.NewApp(tmpDir, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = app.Stop(true, false)
		require.NoError(t, err)
		err = os.Chdir(origDir)
		require.NoError(t, err)
		_ = os.RemoveAll(tmpDir)
	})
	err = app.Start()
	require.NoError(t, err)

	packages := map[string]string{
		"plain": `{"name": "test/plain", "type": "project", "version": "1.0.0"}`,
		"multi": `{"name": "test/multi", "type": "project", "version": "1.0.0", "scripts": {
			"post-root-package-install": ["read -r a && read -r b && echo \"root=$a,$b\" > root.txt"],
			"post-create-project-cmd": ["read -r c && echo \"post=$c\" > post.txt"]
		}}`,
	}
	for name, manifest := range packages {
		err = os.MkdirAll(filepath.Join(tmpDir, ".pkg", name), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, ".pkg", name, "composer.json"), []byte(manifest), 0644)
		require.NoError(t, err)
	}
	createProject := func(composerRoot string, name string, stdin io.Reader) string {
		_, err := exec.RunHostCommand(DdevBin, "config", "--composer-root", composerRoot)
		require.NoError(t, err)
		out, err := exec.RunHostCommandWithOptions(DdevBin, []exec.CmdOption{exec.WithStdin(stdin)}, "composer", "create-project",
			fmt.Sprintf(`--repository={"type":"path","url":"/var/www/html/.pkg/%s","options":{"symlink":false}}`, name), "test/"+name)
		require.NoError(t, err, "output=%s", out)
		return out
	}

	// Stdin stays open for a minute, so a ddev that waits for it fails instead of hanging.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	closeWriter := time.AfterFunc(time.Minute, func() { _ = w.Close() })
	start := time.Now()
	createProject("app1", "plain", r)
	require.Less(t, time.Since(start), time.Minute, "ddev waited for stdin to close")
	closeWriter.Stop()
	_ = w.Close()
	_ = r.Close()

	// No trailing newline: the last answer must still arrive.
	out := createProject("app2", "multi", strings.NewReader("y\ny\ntest"))
	root, err := os.ReadFile(filepath.Join(tmpDir, "app2/root.txt"))
	require.NoError(t, err)
	require.Equal(t, "root=y,y\n", string(root))
	post, err := os.ReadFile(filepath.Join(tmpDir, "app2/post.txt"))
	require.NoError(t, err)
	require.Equal(t, "post=test\n", string(post))
	require.NotRegexp(t, `(?m)^(y|test)\r?$`, out)

	stdout, _, err := app.Exec(&ddevapp.ExecOpts{Cmd: "ls /tmp"})
	require.NoError(t, err)
	require.NotContains(t, stdout, "-input")
}

func TestReadPipedInput(t *testing.T) {
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdin = r
	t.Cleanup(func() {
		_ = os.Stdin.Close()
		os.Stdin = origStdin
		_ = w.Close()
		_ = r.Close()
	})

	pipedInput := readPipedInput()
	_, err = w.Write([]byte("x\n"))
	require.NoError(t, err)
	require.Empty(t, pipedInput())

	require.NoError(t, w.Close())
	require.Eventually(t, func() bool {
		return string(pipedInput()) == "x\n"
	}, 5*time.Second, 10*time.Millisecond)

	n, err := os.Stdin.Read(make([]byte, 1))
	require.Equal(t, 0, n)
	require.ErrorIs(t, err, io.EOF)
}

func TestReadPipedInputLimit(t *testing.T) {
	origStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = origStdin })

	for _, size := range []int{maxPipedInput, maxPipedInput + 1} {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdin = r
		pipedInput := readPipedInput()
		_, err = w.Write([]byte(strings.Repeat("y", size)))
		require.NoError(t, err)
		require.NoError(t, w.Close())

		if size <= maxPipedInput {
			require.Eventually(t, func() bool { return len(pipedInput()) == size }, 5*time.Second, 10*time.Millisecond)
		} else {
			require.Never(t, func() bool { return len(pipedInput()) > 0 }, 500*time.Millisecond, 10*time.Millisecond)
		}
		_ = os.Stdin.Close()
		_ = r.Close()
	}
}

func TestWrapWithPipedInput(t *testing.T) {
	cmd := []string{"composer", "install", "--ignore-platform-req=ext-*", "it's"}
	wrapped, env := wrapWithPipedInput(nil, "/tmp/abc-input", cmd)
	require.Equal(t, cmd, wrapped)
	require.Empty(t, env)

	wrapped, env = wrapWithPipedInput([]byte("x\n"), "/tmp/abc-input", cmd)
	require.Equal(t, []string{"bash", "-c"}, wrapped[:2])
	require.Equal(t, append([]string{"/tmp/abc-input"}, cmd...), wrapped[3:])
	require.Equal(t, []string{"DDEV_PIPED_INPUT=x\n"}, env)
}
