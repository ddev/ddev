package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestUtilityAgentSkillCmd tests ddev utility agent-skill in a configured,
// unstarted project
func TestUtilityAgentSkillCmd(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpXdgConfigHomeDir := testcommon.CopyGlobalDdevDir(t)
	tmpdir := testcommon.CreateTmpDir(t.Name())
	projectName := filepath.Base(tmpdir)
	t.Cleanup(func() {
		_, _ = exec.RunCommand(DdevBin, []string{"delete", "-Oy", projectName})
		_ = os.Chdir(origDir)
		testcommon.CleanupDir(tmpdir)
		testcommon.ResetGlobalDdevDir(t, tmpXdgConfigHomeDir)
	})
	require.NoError(t, os.Chdir(tmpdir))

	_, err := exec.RunCommand(DdevBin, []string{"config", "--docroot=.", "--project-type=php", "--project-name=" + projectName})
	require.NoError(t, err)

	claudeSkill := filepath.Join(tmpdir, ".claude", "skills", "ddev", "SKILL.md")
	agentsSkill := filepath.Join(tmpdir, ".agents", "skills", "ddev", "SKILL.md")

	out, err := exec.RunCommand(DdevBin, []string{"utility", "agent-skill", "install", "--dry-run"})
	require.NoError(t, err, "out=%s", out)
	require.Contains(t, out, "Would be created: "+claudeSkill)
	require.NoFileExists(t, claudeSkill)

	out, err = exec.RunCommand(DdevBin, []string{"utility", "agent-skill", "install"})
	require.NoError(t, err, "out=%s", out)
	require.Contains(t, out, "Created: "+claudeSkill)
	require.Contains(t, out, "Created: "+agentsSkill)
	require.FileExists(t, claudeSkill)
	require.FileExists(t, agentsSkill)

	// A dry run says "Would be" only for files it would change
	out, err = exec.RunCommand(DdevBin, []string{"utility", "agent-skill", "install", "--dry-run"})
	require.NoError(t, err, "out=%s", out)
	require.Contains(t, out, "Unchanged: "+claudeSkill)
	require.NotContains(t, out, "Would be")

	// A changed #ddev-generated copy is replaced
	content, err := os.ReadFile(claudeSkill)
	require.NoError(t, err)
	old := strings.Replace(string(content), "Running that command", "An older DDEV wrote this. Running that command", 1)
	require.NoError(t, os.WriteFile(claudeSkill, []byte(old), 0644))
	out, err = exec.RunCommand(DdevBin, []string{"utility", "agent-skill", "status"})
	require.NoError(t, err, "out=%s", out)
	require.Contains(t, out, claudeSkill+": changed since DDEV")
	out, err = exec.RunCommand(DdevBin, []string{"utility", "agent-skill", "install", "--agent=claude"})
	require.NoError(t, err, "out=%s", out)
	require.Contains(t, out, "Updated")
	updated, err := os.ReadFile(claudeSkill)
	require.NoError(t, err)
	require.Equal(t, string(content), string(updated))

	// An edited copy without the signature is left alone, by install and remove
	edited := strings.ReplaceAll(string(content), nodeps.DdevFileSignature, "edited")
	require.NoError(t, os.WriteFile(agentsSkill, []byte(edited), 0644))
	out, err = exec.RunCommand(DdevBin, []string{"utility", "agent-skill", "install"})
	require.NoError(t, err, "out=%s", out)
	require.Contains(t, out, agentsSkill+": left alone")
	out, err = exec.RunCommand(DdevBin, []string{"utility", "agent-skill", "remove"})
	require.NoError(t, err, "out=%s", out)
	require.Contains(t, out, "Removed: "+claudeSkill)
	require.Contains(t, out, agentsSkill+": left alone")
	// The now-empty .claude directory is removed with the skill
	require.NoDirExists(t, filepath.Join(tmpdir, ".claude"))
	after, err := os.ReadFile(agentsSkill)
	require.NoError(t, err)
	require.Equal(t, edited, string(after))
}
