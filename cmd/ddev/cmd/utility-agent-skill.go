package cmd

import (
	"fmt"
	"strings"

	"github.com/ddev/ddev/pkg/agentskill"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
)

// UtilityAgentSkillCmd is the parent of the agent-skill subcommands
var UtilityAgentSkillCmd = &cobra.Command{
	Use:         "agent-skill",
	Annotations: map[string]string{NoDockerCommand: "true"},
	Short:       "Install, check, or remove the DDEV skill for AI coding agents",
	Long: `Install, check, or remove a skill that tells AI coding agents such as Claude Code
and Codex how to use DDEV: run tools through ddev, which commands only work
on the host, and where to find the docs. DDEV writes the skill only when you
run "ddev utility agent-skill install".

The skill is written to .claude/skills/ddev/SKILL.md (Claude Code) and
.agents/skills/ddev/SKILL.md (Codex and other agents that read .agents/skills),
in the current project or, with --global, in your home directory.`,
	Example: `ddev utility agent-skill install
ddev utility agent-skill install --global
ddev utility agent-skill install --agent=claude --dry-run
ddev utility agent-skill status
ddev utility agent-skill remove`,
	Run: func(cmd *cobra.Command, _ []string) {
		util.CheckErr(cmd.Usage())
	},
}

// UtilityAgentSkillInstallCmd writes or updates the skill
var UtilityAgentSkillInstallCmd = &cobra.Command{
	Use:         "install",
	Annotations: map[string]string{NoDockerCommand: "true"},
	Short:       "Write the DDEV skill for AI coding agents, or update a copy written by an older DDEV",
	Long: `Write the DDEV skill for AI coding agents, or update a copy written by an
older DDEV. A SKILL.md whose #ddev-generated line has been removed is left
alone, so delete that line to keep your own edits.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		files := agentSkillFiles(cmd)
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		version := versionconstants.DdevVersion
		for _, f := range files {
			action, installed, err := agentskill.Install(f, version, dryRun)
			if err != nil {
				util.Failed("Failed to write %s: %v", f, err)
			}
			reportAgentSkillAction(f, action, installed, dryRun)
		}
	},
}

// UtilityAgentSkillStatusCmd reports where the skill is installed
var UtilityAgentSkillStatusCmd = &cobra.Command{
	Use:         "status",
	Annotations: map[string]string{NoDockerCommand: "true"},
	Short:       "Show whether the DDEV skill for AI coding agents is installed and up to date",
	Args:        cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		files := agentSkillFiles(cmd)
		version := versionconstants.DdevVersion
		for _, f := range files {
			status, installed, err := agentskill.Inspect(f, version)
			if err != nil {
				util.Failed("Failed to read %s: %v", f, err)
			}
			switch status {
			case agentskill.Outdated:
				switch installed {
				case "":
					util.Warning("%s: written by an unknown DDEV version, run 'ddev utility agent-skill install' to replace it", f)
				case version:
					util.Warning("%s: changed since DDEV %s wrote it, run 'ddev utility agent-skill install' to replace it", f, installed)
				default:
					util.Warning("%s: written by DDEV %s, run 'ddev utility agent-skill install' to update it for DDEV %s", f, installed, version)
				}
			case agentskill.Edited:
				util.Warning("%s: %s", f, status)
			default:
				output.UserOut.Printf("%s: %s", f, status)
			}
		}
	},
}

// UtilityAgentSkillRemoveCmd removes the skill
var UtilityAgentSkillRemoveCmd = &cobra.Command{
	Use:         "remove",
	Annotations: map[string]string{NoDockerCommand: "true"},
	Short:       "Remove the DDEV skill for AI coding agents; an edited copy is left alone",
	Args:        cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		files := agentSkillFiles(cmd)
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		for _, f := range files {
			action, err := agentskill.Remove(f, dryRun)
			if err != nil {
				util.Failed("Failed to remove %s: %v", f, err)
			}
			reportAgentSkillAction(f, action, "", dryRun)
		}
	},
}

// agentSkillFiles returns the SKILL.md paths selected by --global and --agent
func agentSkillFiles(cmd *cobra.Command) []string {
	global, _ := cmd.Flags().GetBool("global")
	agents, _ := cmd.Flags().GetStringSlice("agent")
	var baseDir string
	if global {
		baseDir = util.GetHomeDir()
	} else {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Run this in a DDEV project, or use --global for the skill in your home directory: %v", err)
		}
		baseDir = app.AppRoot
	}
	files, err := agentskill.SkillFiles(baseDir, agents)
	if err != nil {
		util.Failed("%v", err)
	}
	return files
}

func reportAgentSkillAction(file string, action agentskill.Action, installed string, dryRun bool) {
	verb := strings.ToUpper(string(action[:1])) + string(action[1:])
	if dryRun && (action == agentskill.Created || action == agentskill.Updated || action == agentskill.Removed) {
		verb = "Would be " + string(action)
	}
	switch action {
	case agentskill.Skipped:
		util.Warning("%s: left alone because it has no #ddev-generated line, so it may have your own edits. Delete it, or restore the #ddev-generated line, to let DDEV manage it.", file)
	case agentskill.Updated:
		from := ""
		if installed != "" && installed != versionconstants.DdevVersion {
			from = fmt.Sprintf(" from DDEV %s", installed)
		}
		util.Success("%s%s: %s", verb, from, file)
	case agentskill.Unchanged, agentskill.Absent:
		output.UserOut.Printf("%s: %s", verb, file)
	default:
		util.Success("%s: %s", verb, file)
	}
}

func registerUtilityAgentSkillCmd() {
	for _, c := range []*cobra.Command{UtilityAgentSkillInstallCmd, UtilityAgentSkillStatusCmd, UtilityAgentSkillRemoveCmd} {
		c.Flags().Bool("global", false, "Use the skill directories in your home directory instead of the current project")
		c.Flags().StringSlice("agent", agentskill.Agents, `Which skill directories to use: "claude" for .claude/skills, "agents" for .agents/skills`)
		_ = c.RegisterFlagCompletionFunc("agent", configCompletionFunc(agentskill.Agents))
		UtilityAgentSkillCmd.AddCommand(c)
	}
	for _, c := range []*cobra.Command{UtilityAgentSkillInstallCmd, UtilityAgentSkillRemoveCmd} {
		c.Flags().Bool("dry-run", false, "Show which files would change without writing them")
	}
	DebugCmd.AddCommand(UtilityAgentSkillCmd)
}
