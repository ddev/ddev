package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var (
	downloadDdevPR     int
	downloadDdevBranch string
	downloadDdevCommit string
	downloadDdevTag    string
	downloadDdevStable bool
	downloadDdevHead   bool
	downloadDdevOwner  string
	downloadDdevRepo   string
	downloadDdevOutput string
	downloadDdevOS     string
	downloadDdevArch   string
)

// DownloadDdevCmd implements the "ddev utility download-ddev" command
var DownloadDdevCmd = &cobra.Command{
	Use:         "download-ddev",
	Annotations: map[string]string{NoDockerCommand: "true"},
	Short:       "Download ddev and ddev-hostname binaries built by CI or a release",
	Long: `Download the ddev and ddev-hostname binaries built by DDEV CI for a given
source (PR, branch, commit, release tag, latest stable release, or main HEAD)
and write them into a directory. Exactly one source flag is required.

Releases and main builds are signed; PR and other branch builds are not. No
source needs a GitHub token, but one (DDEV_GITHUB_TOKEN, GH_TOKEN, or
GITHUB_TOKEN) avoids the low anonymous rate limit and downloads CI builds from
GitHub directly rather than through nightly.link. If the GitHub API lookup
fails, --pr falls back to the build links posted on the pull request, and
--head to nightly.link's latest main build, which is sometimes out of date.

By default the binaries are written to ~/tmp/ddev-download-ddev/<version>
rather than the current directory, so a download can't accidentally end up on
PATH or as a stray file in a git checkout, and each build stays separate from
the ones downloaded before it.

This command only downloads the binaries; it does not modify your installed ddev.`,
	Example: `# Download the current platform's build from PR 8611 into ~/tmp/ddev-download-ddev/pr-8611
ddev utility download-ddev --pr 8611

# Download the latest main build into ~/tmp/head
ddev utility download-ddev --head --output ~/tmp/head

# Download a specific released version
ddev utility download-ddev --tag v1.25.3

# Download the latest stable release
ddev utility download-ddev --stable

# Cross-download a macOS arm64 build of a branch
ddev utility download-ddev --branch 20250101_feature --os macos --arch arm64 --output ~/tmp/ddev-macos-arm64`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		if err := runDownloadDdev(cmd); err != nil {
			util.Failed("%v", err)
		}
	},
}

func registerUtilityDownloadDdevCmd() {
	registerDownloadDdevFlags(DownloadDdevCmd)
	DebugCmd.AddCommand(DownloadDdevCmd)
}

// registerDownloadDdevFlags wires up the download-ddev flags and flag groups.
// It is separated from init() so the flag-group rules can be tested on a
// throwaway command.
func registerDownloadDdevFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.IntVar(&downloadDdevPR, "pr", 0, "Pull request number to download the build from")
	f.StringVar(&downloadDdevBranch, "branch", "", "Branch name to download the build from")
	f.StringVar(&downloadDdevCommit, "commit", "", "Commit SHA, full or abbreviated, to download the build from")
	f.StringVar(&downloadDdevTag, "tag", "", "Release tag to download (e.g. v1.25.3)")
	f.BoolVar(&downloadDdevStable, "stable", false, "Download the latest stable release")
	f.BoolVar(&downloadDdevHead, "head", false, "Download the latest main build")
	f.StringVar(&downloadDdevOwner, "owner", "ddev", "GitHub owner/org (for PR builds this is the base repo, not a fork)")
	f.StringVar(&downloadDdevRepo, "repo", "ddev", "GitHub repo")
	f.StringVarP(&downloadDdevOutput, "output", "o", "", "Output directory (default: ~/tmp/ddev-download-ddev/<version>)")
	f.StringVar(&downloadDdevOS, "os", "", "OS override: macos, linux, or windows (default: current OS)")
	f.StringVar(&downloadDdevArch, "arch", "", "Architecture override: amd64 or arm64 (default: current architecture)")

	cmd.MarkFlagsMutuallyExclusive("pr", "branch", "commit", "tag", "stable", "head")
	cmd.MarkFlagsOneRequired("pr", "branch", "commit", "tag", "stable", "head")

	_ = cmd.RegisterFlagCompletionFunc("os", configCompletionFunc([]string{"macos", "linux", "windows"}))
	_ = cmd.RegisterFlagCompletionFunc("arch", configCompletionFunc([]string{"amd64", "arm64"}))
}

// buildTarget describes the OS/arch of the build to download and how its files
// are named.
type buildTarget struct {
	goos     string // runtime-style OS: darwin, linux, or windows
	osName   string // artifact/release naming: macos, linux, or windows
	arch     string // amd64 or arm64
	exeExt   string // "" or ".exe"
	isNative bool   // true when goos/arch match the current machine
}

// artifactName returns the CI artifact name, e.g. "ddev-linux-amd64".
// CI artifacts (uploaded via actions/upload-artifact) use a hyphen after "ddev".
func (t buildTarget) artifactName() string {
	return fmt.Sprintf("ddev-%s-%s", t.osName, t.arch)
}

// releaseAssetName returns the goreleaser release archive name for a tag, e.g.
// "ddev_linux-amd64.v1.25.3.tar.gz". Release archives use an underscore after
// "ddev", embed the tag verbatim (which already includes the leading "v"), and
// are zip files on Windows.
func (t buildTarget) releaseAssetName(tag string) string {
	ext := ".tar.gz"
	if t.goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("ddev_%s-%s.%s%s", t.osName, t.arch, tag, ext)
}

// downloadSpec describes a resolved, ready-to-fetch archive.
type downloadSpec struct {
	url        string
	shaSumURL  string // "" when no checksum file is available (CI artifacts)
	isZip      bool   // true = unzip, false = untar
	signed     bool   // signed/notarized build (releases, main)
	sourceDesc string
	version    string // subdirectory name under the default output dir, e.g. "pr-8611"
}

// runDownloadDdev is the testable entry point for the command.
func runDownloadDdev(cmd *cobra.Command) error {
	target, err := resolveTarget(downloadDdevOS, downloadDdevArch)
	if err != nil {
		return err
	}

	spec, err := resolveSource(cmd, target)
	if err != nil {
		return err
	}

	output.UserOut.Printf("Downloading ddev %s/%s from %s...", target.osName, target.arch, spec.sourceDesc)

	tmpDir, err := os.MkdirTemp("", "ddev-download-ddev-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	archiveName, extract := "download.tar.gz", archive.Untar
	if spec.isZip {
		archiveName, extract = "download.zip", archive.Unzip
	}
	archivePath := filepath.Join(tmpDir, archiveName)
	if err := util.DownloadFile(archivePath, spec.url, true, spec.shaSumURL); err != nil {
		return fmt.Errorf("failed to download %s: %w", spec.url, err)
	}

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := extract(archivePath, extractDir, ""); err != nil {
		return fmt.Errorf("failed to extract downloaded archive: %w", err)
	}

	// The archive also contains mkcert, but we copy only ddev and ddev-hostname:
	// mkcert doesn't change per build, and overwriting a user's mkcert-installed
	// binary is more likely to break trust than to help.
	ddevBin := "ddev" + target.exeExt
	hostnameBin := "ddev-hostname" + target.exeExt
	srcDdev := filepath.Join(extractDir, ddevBin)
	srcHostname := filepath.Join(extractDir, hostnameBin)
	if !fileutil.FileExists(srcDdev) {
		return fmt.Errorf("downloaded archive did not contain %s", ddevBin)
	}

	outputDir := downloadDdevOutput
	if outputDir == "" {
		outputDir = defaultOutputDir(spec.version)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	if abs, absErr := filepath.Abs(outputDir); absErr == nil {
		outputDir = abs
	}

	ddevDest := filepath.Join(outputDir, ddevBin)
	if err := copyExecutable(srcDdev, ddevDest); err != nil {
		return err
	}
	hostnameDest := ""
	if fileutil.FileExists(srcHostname) {
		hostnameDest = filepath.Join(outputDir, hostnameBin)
		if err := copyExecutable(srcHostname, hostnameDest); err != nil {
			return err
		}
	}

	paths := []string{ddevDest}
	if hostnameDest != "" {
		paths = append(paths, hostnameDest)
	}
	autoUnblocked := false
	if !spec.signed {
		switch target.goos {
		case "darwin":
			autoUnblocked = clearMacQuarantine(paths)
		case "windows":
			autoUnblocked = clearWindowsBlock(paths)
		}
	}

	printResult(ddevDest, hostnameDest, target, spec.signed, autoUnblocked)
	return nil
}

// clearMacQuarantine best-effort clears the macOS "com.apple.quarantine" xattr
// from freshly downloaded files, so Gatekeeper doesn't refuse to run them.
// Returns whether it attempted the clear (the xattr tool was found); a missing
// or absent attribute is not treated as a failure, since the file may not
// have been quarantined in the first place.
func clearMacQuarantine(paths []string) bool {
	if _, err := exec.LookPath("xattr"); err != nil {
		return false
	}
	for _, p := range paths {
		_ = exec.Command("xattr", "-d", "com.apple.quarantine", p).Run()
	}
	return true
}

// clearWindowsBlock best-effort clears the Windows "mark of the web" (the
// Zone.Identifier alternate data stream) from freshly downloaded files via
// PowerShell's Unblock-File, so SmartScreen doesn't refuse to run them.
// Returns whether it attempted the clear (PowerShell was found); an absent
// mark is not treated as a failure.
func clearWindowsBlock(paths []string) bool {
	if _, err := exec.LookPath("powershell"); err != nil {
		return false
	}
	for _, p := range paths {
		cmd := fmt.Sprintf("Unblock-File -LiteralPath %s", psQuote(p))
		_ = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd).Run()
	}
	return true
}

// psQuote single-quotes a string for use in a PowerShell command, doubling
// any embedded single quotes per PowerShell's escaping rule.
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// defaultOutputDir returns the download destination used when --output is not
// given. Downloads accumulate here and the set of binaries differs between
// versions (older releases have no ddev-hostname), so each build gets its own
// subdirectory.
func defaultOutputDir(version string) string {
	return filepath.Join(util.GetHomeDir(), "tmp", "ddev-download-ddev", version)
}

// versionDirName makes s safe to use as a single directory name; branch names
// in particular can contain slashes and characters Windows forbids.
func versionDirName(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			return r
		}
		return '-'
	}, s)
}

// resolveTarget maps --os/--arch overrides (or the current machine) to a buildTarget.
func resolveTarget(osFlag, archFlag string) (buildTarget, error) {
	goos := runtime.GOOS
	switch osFlag {
	case "":
		// use runtime.GOOS
	case "macos", "darwin":
		goos = "darwin"
	case "linux", "windows":
		goos = osFlag
	default:
		return buildTarget{}, fmt.Errorf("unsupported --os %q; supported: macos, linux, windows", osFlag)
	}

	arch := runtime.GOARCH
	if archFlag != "" {
		arch = archFlag
	}
	if arch != "amd64" && arch != "arm64" {
		return buildTarget{}, fmt.Errorf("unsupported architecture %q; supported: amd64, arm64", arch)
	}

	var osName, exeExt string
	switch goos {
	case "darwin":
		osName = "macos"
	case "linux":
		osName = "linux"
	case "windows":
		osName, exeExt = "windows", ".exe"
	default:
		return buildTarget{}, fmt.Errorf("unsupported OS %q; supported: macos, linux, windows", goos)
	}

	return buildTarget{
		goos:     goos,
		osName:   osName,
		arch:     arch,
		exeExt:   exeExt,
		isNative: goos == runtime.GOOS && arch == runtime.GOARCH,
	}, nil
}

// resolveSource dispatches on the selected source flag and returns a downloadSpec.
func resolveSource(cmd *cobra.Command, t buildTarget) (downloadSpec, error) {
	owner, repo := downloadDdevOwner, downloadDdevRepo
	f := cmd.Flags()
	switch {
	case f.Changed("tag"):
		if downloadDdevTag == "" {
			return downloadSpec{}, fmt.Errorf("--tag requires a value, e.g. --tag v1.25.3")
		}
		return resolveReleaseTag(owner, repo, downloadDdevTag, t), nil
	case downloadDdevStable:
		return resolveLatestRelease(owner, repo, t)
	case downloadDdevHead:
		return resolveHead(owner, repo, t), nil
	case f.Changed("pr"):
		return resolvePR(owner, repo, downloadDdevPR, t)
	case f.Changed("branch"):
		if downloadDdevBranch == "" {
			return downloadSpec{}, fmt.Errorf("--branch requires a value, e.g. --branch main")
		}
		return resolveBranch(owner, repo, downloadDdevBranch, t)
	case f.Changed("commit"):
		if downloadDdevCommit == "" {
			return downloadSpec{}, fmt.Errorf("--commit requires a commit SHA")
		}
		return resolveCommit(owner, repo, downloadDdevCommit, t)
	}
	return downloadSpec{}, fmt.Errorf("exactly one of --pr, --branch, --commit, --tag, --stable, --head is required")
}

// resolveReleaseTag builds the release-archive URL for an explicit tag without
// any API call; a 404 from the download is the "unknown tag" signal.
func resolveReleaseTag(owner, repo, tag string, t buildTarget) downloadSpec {
	base := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s", owner, repo, tag)
	return downloadSpec{
		url:        base + "/" + t.releaseAssetName(tag),
		shaSumURL:  base + "/checksums.txt",
		isZip:      t.goos == "windows",
		signed:     true,
		sourceDesc: fmt.Sprintf("release %s", tag),
		version:    versionDirName(tag),
	}
}

// resolveLatestRelease discovers the newest release tag, then builds its URL.
func resolveLatestRelease(owner, repo string, t buildTarget) (downloadSpec, error) {
	tag, err := github.GetLatestReleaseTag(owner, repo)
	if err != nil {
		return downloadSpec{}, err
	}
	output.UserOut.Printf("Latest stable release is %s", tag)
	return resolveReleaseTag(owner, repo, tag, t), nil
}

// resolveHead looks up the latest main build through the API, because
// nightly.link's latest-build URL sometimes serves an out-of-date build. That
// URL is only the fallback, such as when the anonymous rate limit is used up.
func resolveHead(owner, repo string, t buildTarget) downloadSpec {
	url, err := resolveArtifactURL(owner, repo, "main-build.yml", t, github.WorkflowRunFilter{Branch: "main"})
	if err != nil {
		util.Warning("Could not find the build through the GitHub API, falling back to nightly.link, whose main build is sometimes out of date: %v", err)
		url = nightlyLinkHeadURL(owner, repo, t)
	}
	return downloadSpec{url: url, isZip: true, signed: true, sourceDesc: "the latest main build", version: "main"}
}

// nightlyLinkHeadURL returns nightly.link's URL for the latest main build,
// which needs no API calls.
func nightlyLinkHeadURL(owner, repo string, t buildTarget) string {
	return fmt.Sprintf("https://nightly.link/%s/%s/workflows/main-build/main/%s.zip", owner, repo, t.artifactName())
}

// resolveArtifactURL looks up a CI artifact and returns its download URL: from
// GitHub when a token is set, and from nightly.link by artifact ID otherwise or
// when that download fails.
func resolveArtifactURL(owner, repo, workflowFile string, t buildTarget, filter github.WorkflowRunFilter) (string, error) {
	id, err := github.FindWorkflowArtifact(owner, repo, workflowFile, t.artifactName(), filter)
	if err != nil {
		return "", err
	}
	if github.HasGitHubToken() {
		url, err := github.ArtifactDownloadURL(owner, repo, id)
		if err == nil {
			return url, nil
		}
		util.Warning("GitHub download of %q failed, falling back to nightly.link: %v", t.artifactName(), err)
	}
	return github.NightlyLinkArtifactURL(owner, repo, id), nil
}

// resolvePR resolves a PR number to its CI artifact. It tries the GitHub API
// first; if that fails (commonly the anonymous rate limit when no token is set),
// it falls back to reading the PR page for the nightly.link URL posted by the
// pr-artifacts-comment bot, which needs no API access. pull_request CI artifacts
// live in the base repo (ddev/ddev) even for fork PRs, so owner/repo stay put.
func resolvePR(owner, repo string, pr int, t buildTarget) (downloadSpec, error) {
	if pr <= 0 {
		return downloadSpec{}, fmt.Errorf("--pr requires a positive PR number")
	}
	spec := downloadSpec{isZip: true, sourceDesc: fmt.Sprintf("PR #%d", pr), version: fmt.Sprintf("pr-%d", pr)}

	url, apiErr := resolvePRViaAPI(owner, repo, pr, t)
	if apiErr == nil {
		spec.url = url
		return spec, nil
	}

	util.Warning("Could not find the build through the GitHub API, falling back to the PR page: %v", apiErr)
	url, pageErr := github.PullRequestArtifactURL(owner, repo, pr, t.artifactName())
	if pageErr != nil {
		return downloadSpec{}, fmt.Errorf("PR #%d: PR-page fallback also failed: %w", pr, pageErr)
	}
	spec.url = url
	return spec, nil
}

// resolvePRViaAPI resolves a PR's artifact download URL through the GitHub API.
func resolvePRViaAPI(owner, repo string, pr int, t buildTarget) (string, error) {
	sha, err := github.GetPullRequestHeadSHA(owner, repo, pr)
	if err != nil {
		return "", err
	}
	url, err := resolveArtifactURL(owner, repo, "pr-build.yml", t, github.WorkflowRunFilter{HeadSHA: sha})
	if err != nil {
		return "", fmt.Errorf("commit %s: %w", shortSHA(sha), err)
	}
	return url, nil
}

// resolveBranch resolves a branch name to its CI artifact. Only main (main-build)
// and branches with an open PR (pr-build) have CI artifacts.
func resolveBranch(owner, repo, branch string, t buildTarget) (downloadSpec, error) {
	workflow, signed := "pr-build.yml", false
	if branch == "main" {
		workflow, signed = "main-build.yml", true
	}
	url, err := resolveArtifactURL(owner, repo, workflow, t, github.WorkflowRunFilter{Branch: branch})
	if errors.Is(err, github.ErrNoSuccessfulRun) {
		err = fmt.Errorf("%w (branch builds exist only for 'main' or branches with an open PR)", err)
	}
	if err != nil {
		return downloadSpec{}, fmt.Errorf("branch %q: %w", branch, err)
	}
	return downloadSpec{url: url, isZip: true, signed: signed, sourceDesc: fmt.Sprintf("branch %s", branch), version: versionDirName("branch-" + branch)}, nil
}

// resolveCommit resolves a commit SHA to its CI artifact, trying PR builds first
// and then the main-branch build.
func resolveCommit(owner, repo, commit string, t buildTarget) (downloadSpec, error) {
	// GitHub's head_sha filter matches only a full SHA.
	if len(commit) < 40 {
		full, err := github.GetCommitSHA(owner, repo, commit)
		if err != nil {
			return downloadSpec{}, err
		}
		commit = full
	}
	signed := false
	url, err := resolveArtifactURL(owner, repo, "pr-build.yml", t, github.WorkflowRunFilter{HeadSHA: commit})
	if errors.Is(err, github.ErrNoSuccessfulRun) {
		signed = true
		url, err = resolveArtifactURL(owner, repo, "main-build.yml", t, github.WorkflowRunFilter{HeadSHA: commit})
		if errors.Is(err, github.ErrNoSuccessfulRun) {
			err = fmt.Errorf("%w for pr-build.yml or main-build.yml", github.ErrNoSuccessfulRun)
		}
	}
	if err != nil {
		return downloadSpec{}, fmt.Errorf("commit %s: %w", shortSHA(commit), err)
	}
	return downloadSpec{url: url, isZip: true, signed: signed, sourceDesc: fmt.Sprintf("commit %s", shortSHA(commit)), version: versionDirName("commit-" + shortSHA(commit))}, nil
}

// copyExecutable copies src to dest and makes it executable.
func copyExecutable(src, dest string) error {
	if err := fileutil.CopyFile(src, dest); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", src, dest, err)
	}
	if err := os.Chmod(dest, 0755); err != nil {
		return fmt.Errorf("failed to make %s executable: %w", dest, err)
	}
	return nil
}

// printResult reports where the binaries landed and how to use them.
// UserOut appends a newline per call, so blank separator lines use Println("").
func printResult(ddevDest, hostnameDest string, t buildTarget, signed, autoUnblocked bool) {
	util.Success("Downloaded ddev to %s", ddevDest)
	if hostnameDest != "" {
		util.Success("Downloaded ddev-hostname to %s", hostnameDest)
	}

	// macOS Gatekeeper and Windows SmartScreen can refuse to run downloaded
	// binaries they consider quarantined/unsigned. main and release builds are
	// signed, but PR and other branch builds are not. runDownloadDdev already
	// tried to clear the quarantine/block automatically; fall back to
	// printing the manual command only if that tool wasn't available.
	if !signed {
		switch t.goos {
		case "darwin":
			if autoUnblocked {
				output.UserOut.Println("")
				output.UserOut.Println("Cleared the macOS quarantine attribute automatically.")
			} else {
				output.UserOut.Println("")
				output.UserOut.Println("On macOS, if the binary is blocked (\"cannot be opened\" or \"killed\"), remove the quarantine attribute:")
				output.UserOut.Printf("  xattr -d com.apple.quarantine %q", ddevDest)
				if hostnameDest != "" {
					output.UserOut.Printf("  xattr -d com.apple.quarantine %q", hostnameDest)
				}
			}
		case "windows":
			if autoUnblocked {
				output.UserOut.Println("")
				output.UserOut.Println("Unblocked the downloaded files automatically (cleared the Windows \"mark of the web\").")
			} else {
				output.UserOut.Println("")
				output.UserOut.Println("On Windows, if the binary is blocked by SmartScreen, unblock it:")
				output.UserOut.Printf("  Unblock-File -LiteralPath %q", ddevDest)
				if hostnameDest != "" {
					output.UserOut.Printf("  Unblock-File -LiteralPath %q", hostnameDest)
				}
			}
		}
	}

	if !t.isNative {
		util.Warning("This is a %s/%s build for another machine (%s/%s); skipping the usage hints.", t.osName, t.arch, runtime.GOOS, runtime.GOARCH)
		return
	}

	output.UserOut.Println("")
	for _, line := range pathHint(filepath.Dir(ddevDest), t.goos, os.Getenv("MSYSTEM")) {
		output.UserOut.Println(line)
	}
}

// pathHint returns the lines telling the user how to put outputDir first on
// PATH in the current shell. `hash -r` clears bash/zsh's cached path to the
// old ddev, which they would otherwise keep using despite the changed PATH.
// On Windows, Git Bash is the shell most DDEV users have, so it comes first,
// and alone when MSYSTEM (set by Git Bash/MSYS2) shows we are running in it.
func pathHint(outputDir, goos, msystem string) []string {
	bash := func(dir, indent string) []string {
		return []string{
			indent + fmt.Sprintf("export PATH=\"%s:$PATH\"", dir),
			indent + "hash -r",
			indent + "ddev version",
		}
	}
	header := "To use this build in the current shell (this window only):"
	if goos != "windows" {
		return append([]string{header}, bash(outputDir, "  ")...)
	}
	gitBashDir := util.WindowsPathToCygwinPath(outputDir)
	if msystem != "" {
		return append([]string{header}, bash(gitBashDir, "  ")...)
	}
	lines := []string{header, "  In Git Bash:"}
	lines = append(lines, bash(gitBashDir, "    ")...)
	return append(lines,
		"  In PowerShell:",
		fmt.Sprintf("    $env:PATH = \"%s;$env:PATH\"", outputDir),
		"    ddev version",
	)
}

// shortSHA returns the first 7 characters of a commit SHA for display.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
