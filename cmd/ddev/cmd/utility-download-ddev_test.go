package cmd

import (
	"archive/zip"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestDownloadDdevResolveTarget verifies OS/arch mapping, .exe suffix, native
// detection, and rejection of unsupported values.
func TestDownloadDdevResolveTarget(t *testing.T) {
	tests := []struct {
		osFlag, archFlag  string
		wantGoos, wantOS  string
		wantArch, wantExe string
		wantErr           bool
	}{
		{"macos", "arm64", "darwin", "macos", "arm64", "", false},
		{"darwin", "amd64", "darwin", "macos", "amd64", "", false},
		{"linux", "amd64", "linux", "linux", "amd64", "", false},
		{"windows", "amd64", "windows", "windows", "amd64", ".exe", false},
		{"windows", "arm64", "windows", "windows", "arm64", ".exe", false},
		{"freebsd", "amd64", "", "", "", "", true},
		{"linux", "386", "", "", "", "", true},
	}
	for _, tc := range tests {
		got, err := resolveTarget(tc.osFlag, tc.archFlag)
		if tc.wantErr {
			require.Error(t, err, "os=%q arch=%q", tc.osFlag, tc.archFlag)
			continue
		}
		require.NoError(t, err, "os=%q arch=%q", tc.osFlag, tc.archFlag)
		require.Equal(t, tc.wantGoos, got.goos)
		require.Equal(t, tc.wantOS, got.osName)
		require.Equal(t, tc.wantArch, got.arch)
		require.Equal(t, tc.wantExe, got.exeExt)
	}

	// Defaults should match the current machine and be marked native.
	native, err := resolveTarget("", "")
	require.NoError(t, err)
	require.Equal(t, runtime.GOOS, native.goos)
	require.Equal(t, runtime.GOARCH, native.arch)
	require.True(t, native.isNative, "no overrides should produce a native target")
}

// TestDownloadDdevArtifactName locks down the CI artifact naming (hyphen after ddev).
func TestDownloadDdevArtifactName(t *testing.T) {
	require.Equal(t, "ddev-linux-amd64", buildTarget{osName: "linux", arch: "amd64"}.artifactName())
	require.Equal(t, "ddev-macos-arm64", buildTarget{osName: "macos", arch: "arm64"}.artifactName())
	require.Equal(t, "ddev-windows-amd64", buildTarget{osName: "windows", arch: "amd64"}.artifactName())
}

// TestDownloadDdevReleaseAssetName locks down goreleaser release naming: underscore
// after ddev, the tag embedded verbatim (already includes the "v"), and .zip on
// Windows.
func TestDownloadDdevReleaseAssetName(t *testing.T) {
	require.Equal(t, "ddev_linux-amd64.v1.24.5.tar.gz",
		buildTarget{goos: "linux", osName: "linux", arch: "amd64"}.releaseAssetName("v1.24.5"))
	require.Equal(t, "ddev_macos-arm64.v1.24.5.tar.gz",
		buildTarget{goos: "darwin", osName: "macos", arch: "arm64"}.releaseAssetName("v1.24.5"))
	require.Equal(t, "ddev_windows-amd64.v1.24.5.zip",
		buildTarget{goos: "windows", osName: "windows", arch: "amd64"}.releaseAssetName("v1.24.5"))
}

// TestDownloadDdevURLs verifies the nightly.link and release URLs that are built
// without any network access.
func TestDownloadDdevURLs(t *testing.T) {
	linux := buildTarget{goos: "linux", osName: "linux", arch: "amd64"}

	head := resolveHead("ddev", "ddev", linux)
	require.Equal(t, "https://nightly.link/ddev/ddev/workflows/main-build/main/ddev-linux-amd64.zip", head.url)
	require.True(t, head.isZip)
	require.Empty(t, head.shaSumURL)

	require.Equal(t, "https://nightly.link/ddev/ddev/actions/artifacts/12345.zip",
		github.NightlyLinkArtifactURL("ddev", "ddev", 12345))

	rel := resolveReleaseTag("ddev", "ddev", "v1.24.5", linux)
	require.Equal(t, "https://github.com/ddev/ddev/releases/download/v1.24.5/ddev_linux-amd64.v1.24.5.tar.gz", rel.url)
	require.Equal(t, "https://github.com/ddev/ddev/releases/download/v1.24.5/checksums.txt", rel.shaSumURL)
	require.False(t, rel.isZip)

	// Windows release uses a zip archive.
	win := buildTarget{goos: "windows", osName: "windows", arch: "amd64", exeExt: ".exe"}
	relWin := resolveReleaseTag("ddev", "ddev", "v1.24.5", win)
	require.Equal(t, "https://github.com/ddev/ddev/releases/download/v1.24.5/ddev_windows-amd64.v1.24.5.zip", relWin.url)
	require.True(t, relWin.isZip)
}

// TestDownloadDdevExtractAndCopy exercises the real unzip + copyExecutable path used
// after a download: a flat artifact zip is extracted and its ddev binary is
// copied into the output directory with an executable mode.
func TestDownloadDdevExtractAndCopy(t *testing.T) {
	tmp := t.TempDir()

	// Build a flat artifact zip like the ones CI uploads.
	zipPath := filepath.Join(tmp, "ddev-linux-amd64.zip")
	zf, err := os.Create(zipPath)
	require.NoError(t, err)
	zw := zip.NewWriter(zf)
	for _, name := range []string{"ddev", "ddev-hostname", "mkcert"} {
		w, werr := zw.Create(name)
		require.NoError(t, werr)
		_, werr = w.Write([]byte("#!/bin/sh\necho " + name + "\n"))
		require.NoError(t, werr)
	}
	require.NoError(t, zw.Close())
	require.NoError(t, zf.Close())

	extractDir := filepath.Join(tmp, "extracted")
	require.NoError(t, archive.Unzip(zipPath, extractDir, ""))
	require.FileExists(t, filepath.Join(extractDir, "ddev"))
	require.FileExists(t, filepath.Join(extractDir, "ddev-hostname"))

	outDir := filepath.Join(tmp, "out")
	require.NoError(t, os.MkdirAll(outDir, 0755))

	dest := filepath.Join(outDir, "ddev")
	require.NoError(t, copyExecutable(filepath.Join(extractDir, "ddev"), dest))
	require.FileExists(t, dest)

	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(dest)
		require.NoError(t, statErr)
		require.NotZero(t, info.Mode()&0111, "copied binary must be executable")
	}
}

// TestDownloadDdevFlagGroups verifies the mutually-exclusive / one-required source
// selectors on a throwaway command (so the shared command's flag state is not
// mutated).
func TestDownloadDdevFlagGroups(t *testing.T) {
	newCmd := func() *cobra.Command {
		c := &cobra.Command{Use: "download-ddev", Run: func(_ *cobra.Command, _ []string) {}}
		c.SilenceUsage = true
		c.SilenceErrors = true
		registerDownloadDdevFlags(c)
		return c
	}

	// No source selector -> error (one required).
	none := newCmd()
	none.SetArgs([]string{})
	require.Error(t, none.Execute(), "no source selector should be rejected")

	// Two source selectors -> error (mutually exclusive).
	two := newCmd()
	two.SetArgs([]string{"--pr", "1", "--head"})
	require.Error(t, two.Execute(), "two source selectors should be rejected")

	// A single selector passes flag validation (the no-op Run does nothing).
	one := newCmd()
	one.SetArgs([]string{"--head"})
	require.NoError(t, one.Execute(), "a single source selector should pass validation")

	// A --tag value passes flag validation.
	tag := newCmd()
	tag.SetArgs([]string{"--tag", "v1.2.3"})
	require.NoError(t, tag.Execute(), "--tag should be accepted")
}

// TestDownloadDdevHeadDownload is a live, opt-in integration test (gated on
// DDEV_RUN_DOWNLOAD_DDEV_TEST, which the nginx-fpm CI job sets) that runs the
// real `ddev utility download-ddev --head` and then runs the downloaded binary.
//
// It exercises only --head: it always resolves (main is built continuously),
// needs no token or API budget (plain nightly.link), and covers the shared
// download → extract → verify → copy pipeline that every source funnels into.
// PR/branch/commit artifacts expire after ~90 days, so hardcoding one would rot;
// their URL logic is already covered offline by TestDownloadDdevURLs above.
func TestDownloadDdevHeadDownload(t *testing.T) {
	if nodeps.IsEnvFalse("DDEV_RUN_DOWNLOAD_DDEV_TEST") {
		t.Skip("Skip live download test unless DDEV_RUN_DOWNLOAD_DDEV_TEST=true")
	}

	outDir := testcommon.CreateTmpDir(filepath.Base(t.Name()))
	defer testcommon.CleanupDir(outDir)

	out, err := exec.RunHostCommand(DdevBin, "utility", "download-ddev", "--head", "--output", outDir)
	require.NoError(t, err, "download-ddev --head failed, out='%s'", out)

	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}
	ddevPath := filepath.Join(outDir, "ddev"+exe)
	require.FileExists(t, ddevPath)
	require.FileExists(t, filepath.Join(outDir, "ddev-hostname"+exe))

	// --head downloads the current machine's build, so the copied binary must run.
	out, err = exec.RunHostCommand(ddevPath, "version")
	require.NoError(t, err, "downloaded ddev failed to run, out='%s'", out)
	require.Contains(t, out, "DDEV version")
}
