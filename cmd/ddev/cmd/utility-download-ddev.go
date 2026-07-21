package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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
	Use:   "download-ddev",
	Short: "Download ddev and ddev-hostname binaries built by CI or a release",
	Long: `Download the ddev and ddev-hostname binaries built by DDEV CI for a given
source (PR, branch, commit, release tag, latest stable release, or main HEAD)
and write them into a directory. Exactly one source flag is required.

--tag, --stable, and --head download signed binaries and need no GitHub token
(releases from github.com, the main build from nightly.link). --pr, --branch,
and --commit download unsigned GitHub Actions artifacts and use a GitHub token
(DDEV_GITHUB_TOKEN, GH_TOKEN, or GITHUB_TOKEN) if one is set to avoid the low
anonymous rate limit; without a token, --pr falls back to the build links
posted on the pull request.

This command only downloads the binaries; it does not modify your installed ddev.`,
	Example: `# Download the current platform's build from PR 1234 into the current directory
ddev utility download-ddev --pr 1234

# Download the latest main build into ~/tmp/head
ddev utility download-ddev --head --output ~/tmp/head

# Download a specific released version
ddev utility download-ddev --tag v1.24.5

# Download the latest stable release
ddev utility download-ddev --stable

# Cross-download a macOS arm64 build of a branch
ddev utility download-ddev --branch 20250101_feature --os macos --arch arm64 --output ./out`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		if err := runDownloadDdev(cmd); err != nil {
			util.Failed("%v", err)
		}
	},
}

func init() {
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
	f.StringVar(&downloadDdevCommit, "commit", "", "Commit SHA to download the build from")
	f.StringVar(&downloadDdevTag, "tag", "", "Release tag to download (e.g. v1.24.5)")
	f.BoolVar(&downloadDdevStable, "stable", false, "Download the latest stable release")
	f.BoolVar(&downloadDdevHead, "head", false, "Download the latest main build")
	f.StringVar(&downloadDdevOwner, "owner", "ddev", "GitHub owner/org (for PR builds this is the base repo, not a fork)")
	f.StringVar(&downloadDdevRepo, "repo", "ddev", "GitHub repo")
	f.StringVarP(&downloadDdevOutput, "output", "o", "", "Output directory (default: current directory)")
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
// "ddev_linux-amd64.v1.24.5.tar.gz". Release archives use an underscore after
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
		if outputDir, err = os.Getwd(); err != nil {
			return err
		}
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

	printResult(ddevDest, hostnameDest, target, spec.signed)
	return nil
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
			return downloadSpec{}, fmt.Errorf("--tag requires a value, e.g. --tag v1.24.5")
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

// resolveHead builds the nightly.link URL for the latest main build. nightly.link
// serves the most recent main-build artifact without a GitHub token.
func resolveHead(owner, repo string, t buildTarget) downloadSpec {
	return downloadSpec{
		url:        fmt.Sprintf("https://nightly.link/%s/%s/workflows/main-build/main/%s.zip", owner, repo, t.artifactName()),
		isZip:      true,
		signed:     true,
		sourceDesc: "the latest main build",
	}
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
	spec := downloadSpec{isZip: true, sourceDesc: fmt.Sprintf("PR #%d", pr)}

	url, apiErr := resolvePRViaAPI(owner, repo, pr, t)
	if apiErr == nil {
		spec.url = url
		return spec, nil
	}

	// API failed; fall back to the token-free PR page.
	output.UserOut.Println("GitHub API lookup failed; falling back to the PR page.")
	url, pageErr := github.PullRequestArtifactURL(owner, repo, pr, t.artifactName())
	if pageErr != nil {
		return downloadSpec{}, fmt.Errorf("PR #%d: %w\nPR-page fallback also failed: %v", pr, apiErr, pageErr)
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
	url, err := github.ResolveWorkflowArtifactURL(owner, repo, "pr-build.yml", t.artifactName(), github.WorkflowRunFilter{HeadSHA: sha, Event: "pull_request"})
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
	url, err := github.ResolveWorkflowArtifactURL(owner, repo, workflow, t.artifactName(), github.WorkflowRunFilter{Branch: branch})
	if err != nil {
		return downloadSpec{}, fmt.Errorf("branch %q: %w (branch builds exist only for 'main' or branches with an open PR)", branch, err)
	}
	return downloadSpec{url: url, isZip: true, signed: signed, sourceDesc: fmt.Sprintf("branch %s", branch)}, nil
}

// resolveCommit resolves a commit SHA to its CI artifact, trying PR builds first
// and then the main-branch build.
func resolveCommit(owner, repo, commit string, t buildTarget) (downloadSpec, error) {
	signed := false
	url, err := github.ResolveWorkflowArtifactURL(owner, repo, "pr-build.yml", t.artifactName(), github.WorkflowRunFilter{HeadSHA: commit})
	if err != nil {
		var mainErr error
		url, mainErr = github.ResolveWorkflowArtifactURL(owner, repo, "main-build.yml", t.artifactName(), github.WorkflowRunFilter{HeadSHA: commit})
		if mainErr != nil {
			return downloadSpec{}, fmt.Errorf("commit %s: %w", shortSHA(commit), err)
		}
		signed = true
	}
	return downloadSpec{url: url, isZip: true, signed: signed, sourceDesc: fmt.Sprintf("commit %s", shortSHA(commit))}, nil
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

// printResult reports where the binaries landed and how to use or install them.
// UserOut appends a newline per call, so blank separator lines use Println("").
func printResult(ddevDest, hostnameDest string, t buildTarget, signed bool) {
	util.Success("Downloaded ddev to %s", ddevDest)
	if hostnameDest != "" {
		util.Success("Downloaded ddev-hostname to %s", hostnameDest)
	}

	// macOS Gatekeeper refuses to run downloaded binaries it considers
	// quarantined or unsigned. main-branch and released builds are signed, but
	// PR/branch/commit builds are not, so tell macOS users how to clear the
	// quarantine if the binary won't start.
	if t.goos == "darwin" && !signed {
		output.UserOut.Println("")
		output.UserOut.Println("On macOS, if the binary is blocked (\"cannot be opened\" or \"killed\"), remove the quarantine attribute:")
		output.UserOut.Printf("  xattr -r -d com.apple.quarantine %q", ddevDest)
		if hostnameDest != "" {
			output.UserOut.Printf("  xattr -r -d com.apple.quarantine %q", hostnameDest)
		}
	}

	if !t.isNative {
		util.Warning("This is a %s/%s build for another machine (%s/%s); skipping the install hints.", t.osName, t.arch, runtime.GOOS, runtime.GOARCH)
		return
	}

	outputDir := filepath.Dir(ddevDest)

	// Temporary: prepend the download directory so `ddev` in the current shell
	// resolves to it. `hash -r` clears the shell's cached path to the old ddev,
	// which bash/zsh would otherwise keep using despite the changed PATH.
	output.UserOut.Println("")
	output.UserOut.Println("To use this build in the current shell (this window only):")
	if t.goos == "windows" {
		output.UserOut.Printf("  $env:PATH = \"%s;$env:PATH\"", outputDir)
	} else {
		output.UserOut.Printf("  export PATH=\"%s:$PATH\"", outputDir)
		output.UserOut.Println("  hash -r")
	}
	output.UserOut.Println("  ddev version")

	// Permanent (Unix): replace the binary that `ddev` currently resolves to on
	// PATH, following a symlink (e.g. Homebrew) to the real file so we replace
	// the destination, not the link. Back up each current binary to <path>.bak
	// first so the change can be reverted.
	if t.goos == "windows" {
		return
	}
	curDdev, ddevWasLink := currentBinaryPath("ddev")
	if curDdev == "" {
		return
	}

	// Pair each on-PATH binary with its freshly downloaded replacement.
	type replacement struct{ cur, src string }
	replacements := []replacement{{cur: curDdev, src: ddevDest}}
	if hostnameDest != "" {
		if curHostname, _ := currentBinaryPath("ddev-hostname"); curHostname != "" {
			replacements = append(replacements, replacement{cur: curHostname, src: hostnameDest})
		}
	}

	sudo := ""
	if !isDirWritable(filepath.Dir(curDdev)) {
		sudo = "sudo "
	}

	output.UserOut.Println("")
	output.UserOut.Println("To install this build permanently, back up your current binary and replace it:")
	for _, r := range replacements {
		output.UserOut.Printf("  %scp %q %q", sudo, r.cur, r.cur+".bak")
		output.UserOut.Printf("  %scp %q %q", sudo, r.src, r.cur)
	}
	if ddevWasLink {
		output.UserOut.Println("  (ddev on your PATH is a symlink; the path above is its real target)")
	}

	output.UserOut.Println("")
	output.UserOut.Println("To revert to your previous binary:")
	for _, r := range replacements {
		output.UserOut.Printf("  %smv %q %q", sudo, r.cur+".bak", r.cur)
	}
}

// currentBinaryPath returns the real path of the named binary as found on PATH,
// resolving any symlink to its target. The bool reports whether a symlink was
// resolved. Returns ("", false) when the binary is not on PATH.
func currentBinaryPath(name string) (string, bool) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p, false
	}
	return resolved, resolved != p
}

// isDirWritable reports whether a file can be created in dir.
func isDirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".ddev-write-test-")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// shortSHA returns the first 7 characters of a commit SHA for display.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
