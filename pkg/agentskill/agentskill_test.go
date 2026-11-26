package agentskill_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/agentskill"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

func TestSkillFiles(t *testing.T) {
	files, err := agentskill.SkillFiles("/base", agentskill.Agents)
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join("/base", ".claude", "skills", "ddev", "SKILL.md"),
		filepath.Join("/base", ".agents", "skills", "ddev", "SKILL.md"),
	}, files)

	files, err = agentskill.SkillFiles("/base", []string{"claude", "claude"})
	require.NoError(t, err)
	require.Len(t, files, 1)

	_, err = agentskill.SkillFiles("/base", []string{"cursor"})
	require.ErrorContains(t, err, `unknown agent "cursor"`)
}

func TestDocsURL(t *testing.T) {
	require.Equal(t, "https://docs.ddev.com/en/stable", agentskill.DocsURL("v1.25.5"))
	require.Equal(t, "https://docs.ddev.com/en/latest", agentskill.DocsURL("v1.26.0-rc1"))
	require.Equal(t, "https://docs.ddev.com/en/latest", agentskill.DocsURL("v1.25.5-87-gf8151770f"))
	require.Equal(t, "https://docs.ddev.com/en/latest", agentskill.DocsURL("v1.25.5-dirty"))
}

func TestRender(t *testing.T) {
	content := agentskill.Render("v1.25.5")
	// Agent Skills frontmatter must start on the first line
	require.True(t, strings.HasPrefix(content, "---\nname: ddev\ndescription: "))
	require.Contains(t, content, nodeps.DdevFileSignature)
	require.Contains(t, content, "from DDEV v1.25.5.")
	require.Contains(t, content, "https://docs.ddev.com/en/stable/llms.txt")
	require.NotContains(t, content, "@@")
}

func TestInstallUpdateRemove(t *testing.T) {
	base := t.TempDir()
	skill := filepath.Join(base, ".claude", "skills", "ddev", "SKILL.md")

	status, _, err := agentskill.Inspect(skill, "v1.25.5")
	require.NoError(t, err)
	require.Equal(t, agentskill.NotInstalled, status)

	action, _, err := agentskill.Install(skill, "v1.25.5", true)
	require.NoError(t, err)
	require.Equal(t, agentskill.Created, action)
	require.NoFileExists(t, skill, "dry run must not write")

	action, _, err = agentskill.Install(skill, "v1.25.5", false)
	require.NoError(t, err)
	require.Equal(t, agentskill.Created, action)
	require.FileExists(t, skill)

	action, _, err = agentskill.Install(skill, "v1.25.5", false)
	require.NoError(t, err)
	require.Equal(t, agentskill.Unchanged, action)

	// A newer DDEV updates a #ddev-generated copy
	status, installed, err := agentskill.Inspect(skill, "v1.26.0")
	require.NoError(t, err)
	require.Equal(t, agentskill.Outdated, status)
	require.Equal(t, "v1.25.5", installed)
	action, installed, err = agentskill.Install(skill, "v1.26.0", false)
	require.NoError(t, err)
	require.Equal(t, agentskill.Updated, action)
	require.Equal(t, "v1.25.5", installed)
	content, err := os.ReadFile(skill)
	require.NoError(t, err)
	require.Contains(t, string(content), "from DDEV v1.26.0.")

	action, err = agentskill.Remove(skill, true)
	require.NoError(t, err)
	require.Equal(t, agentskill.Removed, action)
	require.FileExists(t, skill, "dry run must not remove")

	action, err = agentskill.Remove(skill, false)
	require.NoError(t, err)
	require.Equal(t, agentskill.Removed, action)
	// The now-empty skill, skills, and .claude directories go too
	require.NoDirExists(t, filepath.Join(base, ".claude"))
	require.DirExists(t, base)

	action, err = agentskill.Remove(skill, false)
	require.NoError(t, err)
	require.Equal(t, agentskill.Absent, action)
}

func TestEditedSkillIsLeftAlone(t *testing.T) {
	skill := filepath.Join(t.TempDir(), ".agents", "skills", "ddev", "SKILL.md")
	_, _, err := agentskill.Install(skill, "v1.25.5", false)
	require.NoError(t, err)

	// Removing the #ddev-generated line marks the file as the user's own
	content, err := os.ReadFile(skill)
	require.NoError(t, err)
	var kept []string
	for line := range strings.SplitSeq(string(content), "\n") {
		if !strings.Contains(line, nodeps.DdevFileSignature) {
			kept = append(kept, line)
		}
	}
	edited := strings.Join(kept, "\n") + "\nMy own notes.\n"
	require.NoError(t, os.WriteFile(skill, []byte(edited), 0644))

	status, _, err := agentskill.Inspect(skill, "v1.26.0")
	require.NoError(t, err)
	require.Equal(t, agentskill.Edited, status)

	action, _, err := agentskill.Install(skill, "v1.26.0", false)
	require.NoError(t, err)
	require.Equal(t, agentskill.Skipped, action)

	action, err = agentskill.Remove(skill, false)
	require.NoError(t, err)
	require.Equal(t, agentskill.Skipped, action)

	after, err := os.ReadFile(skill)
	require.NoError(t, err)
	require.Equal(t, edited, string(after))
}

func TestRemoveKeepsNonEmptyDirs(t *testing.T) {
	base := t.TempDir()
	skill := filepath.Join(base, ".claude", "skills", "ddev", "SKILL.md")
	other := filepath.Join(base, ".claude", "settings.json")
	_, _, err := agentskill.Install(skill, "v1.25.5", false)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(other, []byte("{}"), 0644))

	action, err := agentskill.Remove(skill, false)
	require.NoError(t, err)
	require.Equal(t, agentskill.Removed, action)
	require.NoDirExists(t, filepath.Join(base, ".claude", "skills"))
	require.FileExists(t, other)
}

func TestRemoveKeepsSymlinkedDirs(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("creating symlinks needs extra privileges on Windows")
	}

	// .claude/skills -> ../.agents/skills, with another skill beside ddev
	base := t.TempDir()
	other := filepath.Join(base, ".agents", "skills", "other")
	require.NoError(t, os.MkdirAll(other, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(base, ".claude"), 0755))
	require.NoError(t, os.Symlink(filepath.Join("..", ".agents", "skills"), filepath.Join(base, ".claude", "skills")))
	skill := filepath.Join(base, ".claude", "skills", "ddev", "SKILL.md")
	_, _, err := agentskill.Install(skill, "v1.25.5", false)
	require.NoError(t, err)

	action, err := agentskill.Remove(skill, false)
	require.NoError(t, err)
	require.Equal(t, agentskill.Removed, action)
	requireSymlink(t, filepath.Join(base, ".claude", "skills"))
	require.DirExists(t, other)

	// A symlinked .claude, as when it lives in a dotfiles repo
	base = t.TempDir()
	target := filepath.Join(base, "dotfiles-claude")
	require.NoError(t, os.MkdirAll(target, 0755))
	require.NoError(t, os.Symlink(target, filepath.Join(base, ".claude")))
	skill = filepath.Join(base, ".claude", "skills", "ddev", "SKILL.md")
	_, _, err = agentskill.Install(skill, "v1.25.5", false)
	require.NoError(t, err)

	action, err = agentskill.Remove(skill, false)
	require.NoError(t, err)
	require.Equal(t, agentskill.Removed, action)
	requireSymlink(t, filepath.Join(base, ".claude"))
	require.DirExists(t, target)
}

func requireSymlink(t *testing.T, path string) {
	t.Helper()
	fi, err := os.Lstat(path)
	require.NoError(t, err)
	require.Equal(t, os.ModeSymlink, fi.Mode()&os.ModeSymlink, "%s should still be a symlink", path)
}
