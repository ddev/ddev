// Package agentskill writes and removes the DDEV skill for AI coding agents,
// in the Agent Skills format (https://agentskills.io). DDEV writes the skill
// only when the user runs `ddev utility agent-skill install`.
package agentskill

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/versionconstants"
)

//go:embed skill/SKILL.md
var skillTemplate string

// SkillName is the skill's name and the name of its directory.
const SkillName = "ddev"

// Agent families and the directory each one reads skills from, relative to
// the project root or the home directory.
const (
	AgentClaude = "claude"
	AgentAgents = "agents"
)

var agentDirs = map[string]string{
	AgentClaude: filepath.Join(".claude", "skills"),
	AgentAgents: filepath.Join(".agents", "skills"),
}

// Agents lists the valid agent names in a stable order.
var Agents = []string{AgentClaude, AgentAgents}

var versionPattern = regexp.MustCompile(nodeps.DdevFileSignature + `: written by .* from DDEV (\S+)\.`)

// Status describes an existing skill file.
type Status int

const (
	// NotInstalled means there is no skill file.
	NotInstalled Status = iota
	// Current means a #ddev-generated skill file matches this DDEV version.
	Current
	// Outdated means a #ddev-generated skill file came from another DDEV version.
	Outdated
	// Edited means the file has no #ddev-generated signature, so DDEV leaves it alone.
	Edited
)

func (s Status) String() string {
	switch s {
	case Current:
		return "current"
	case Outdated:
		return "different DDEV version"
	case Edited:
		return "edited, not managed by DDEV"
	default:
		return "not installed"
	}
}

// Action is what Install or Remove did, or would do with dryRun.
type Action string

// Actions reported by Install and Remove.
const (
	Created   Action = "created"
	Updated   Action = "updated"
	Unchanged Action = "unchanged"
	Removed   Action = "removed"
	Skipped   Action = "skipped"
	Absent    Action = "not installed"
)

// SkillFiles returns the SKILL.md path for each agent under baseDir, which
// is a project root or the home directory.
func SkillFiles(baseDir string, agents []string) ([]string, error) {
	var files []string
	for _, a := range agents {
		dir, ok := agentDirs[a]
		if !ok {
			return nil, fmt.Errorf("unknown agent %q, use one of: %s", a, strings.Join(Agents, ", "))
		}
		f := filepath.Join(baseDir, dir, SkillName, "SKILL.md")
		if !slices.Contains(files, f) {
			files = append(files, f)
		}
	}
	return files, nil
}

// DocsURL returns the docs base URL closest to version. Per-release docs
// don't publish llms.txt, so a release uses stable and anything else latest.
func DocsURL(version string) string {
	if strings.HasPrefix(version, "v") && !strings.Contains(version, "-") && !versionconstants.IsUnreleasedDdevVersion(version) {
		return "https://docs.ddev.com/en/stable"
	}
	return "https://docs.ddev.com/en/latest"
}

// Render returns the SKILL.md content for a DDEV version.
func Render(version string) string {
	return strings.NewReplacer(
		"@@DDEV_VERSION@@", version,
		"@@DOCS_URL@@", DocsURL(version),
	).Replace(skillTemplate)
}

// Inspect reports the status of the skill file at path and, for a
// #ddev-generated file, the DDEV version that wrote it.
func Inspect(path string, version string) (Status, string, error) {
	if !fileutil.FileExists(path) {
		return NotInstalled, "", nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return NotInstalled, "", err
	}
	if !strings.Contains(string(content), nodeps.DdevFileSignature) {
		if len(content) == 0 {
			return Outdated, "", nil
		}
		return Edited, "", nil
	}
	installed := ""
	if m := versionPattern.FindStringSubmatch(string(content)); m != nil {
		installed = m[1]
	}
	if string(content) == Render(version) {
		return Current, installed, nil
	}
	return Outdated, installed, nil
}

// Install writes the skill for version to path unless the file there has
// been edited. With dryRun it reports what it would do without writing.
func Install(path string, version string, dryRun bool) (Action, error) {
	status, _, err := Inspect(path, version)
	if err != nil {
		return "", err
	}
	action := Updated
	switch status {
	case Current:
		return Unchanged, nil
	case Edited:
		return Skipped, nil
	case NotInstalled:
		action = Created
	}
	if dryRun {
		return action, nil
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err = os.WriteFile(path, []byte(Render(version)), 0644); err != nil {
		return "", err
	}
	return action, nil
}

// Remove deletes a #ddev-generated skill file at path, and its directory if
// that is then empty. An edited file is left alone.
func Remove(path string, dryRun bool) (Action, error) {
	status, _, err := Inspect(path, "")
	if err != nil {
		return "", err
	}
	switch status {
	case NotInstalled:
		return Absent, nil
	case Edited:
		return Skipped, nil
	}
	if dryRun {
		return Removed, nil
	}
	if err = os.Remove(path); err != nil {
		return "", err
	}
	// Fails harmlessly when the user added other files to the skill directory
	_ = os.Remove(filepath.Dir(path))
	return Removed, nil
}
