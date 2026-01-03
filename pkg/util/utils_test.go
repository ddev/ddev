package util_test

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRandString ensures that RandString only generates string of the correct value and characters.
func TestRandString(t *testing.T) {
	assert := asrt.New(t)
	stringLengths := []int{2, 4, 8, 16, 23, 47}

	for _, stringLength := range stringLengths {
		testString := util.RandString(stringLength)
		assert.Equal(len(testString), stringLength, fmt.Sprintf("Generated string is of length %d", stringLengths))
	}
	lb := "a"
	util.SetLetterBytes(lb)
	testString := util.RandString(1)
	assert.Equal(testString, lb)
}

// TestGetInput tests GetInput and Prompt()
func TestGetInput(t *testing.T) {
	assert := asrt.New(t)

	// Try basic GetInput
	input := "InputIWantToSee"
	restoreOutput := util.CaptureUserOut()
	scanner := bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)
	result := util.GetInput("nodefault")
	assert.EqualValues(input, result)
	_ = restoreOutput()

	// Try Prompt() with a default value which is overridden
	input = "InputIWantToSee"
	restoreOutput = util.CaptureUserOut()
	scanner = bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)
	result = util.Prompt("nodefault", "expected default")
	assert.EqualValues(input, result)
	_ = restoreOutput()

	// Try Prompt() with a default value but don't provide a response
	input = ""
	restoreOutput = util.CaptureUserOut()
	scanner = bufio.NewScanner(strings.NewReader(input))
	util.SetInputScanner(scanner)
	result = util.Prompt("nodefault", "expected default")
	assert.EqualValues("expected default", result)
	_ = restoreOutput()
	println() // Just lets goland find the PASS or FAIL
}

// TestGetQuotedInput tests GetQuotedInput
func TestGetQuotedInput(t *testing.T) {
	testCases := []struct {
		description string
		input       string
		expected    string
	}{
		{"Remove single quotes", `'/path/to/file'`, `/path/to/file`},
		{"Remove double quotes", `"/path/to/file"`, `/path/to/file`},
		{"Remove spaces", `  /path/to/file  `, `/path/to/file`},
		{"Remove all quotes and spaces", `  ''' """ /path/to/file '''  """ `, `/path/to/file`},
		{"Quotes and spaces are not removed from the middle", `/path/'" to/file`, `/path/'" to/file`},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert := asrt.New(t)
			restoreOutput := util.CaptureUserOut()
			scanner := bufio.NewScanner(strings.NewReader(tc.input))
			util.SetInputScanner(scanner)
			result := util.GetQuotedInput("nodefault")
			assert.EqualValues(tc.expected, result)
			_ = restoreOutput()
		})
	}
}

// TestCaptureUserOut ensures capturing of stdout works as expected.
func TestCaptureUserOut(t *testing.T) {
	assert := asrt.New(t)
	restoreOutput := util.CaptureUserOut()
	text := util.RandString(128)
	output.UserOut.Println(text)
	out := restoreOutput()

	assert.Contains(out, text)
}

// TestCaptureStdOut ensures capturing of stdout works as expected.
func TestCaptureStdOut(t *testing.T) {
	assert := asrt.New(t)
	restoreOutput := util.CaptureStdOut()
	text := util.RandString(128)
	fmt.Println(text)
	out := restoreOutput()

	assert.Contains(out, text)
}

// TestCaptureOutputToFile tests CaptureOutputToFile.
func TestCaptureOutputToFile(t *testing.T) {
	assert := asrt.New(t)
	restoreOutput, err := util.CaptureOutputToFile()
	assert.NoError(err)
	text := util.RandString(128)
	fmt.Println("randstring-println=" + text)
	c := exec.Command("sh", "-c", fmt.Sprintf("echo randstring-stdout=%s; echo 1>&2 randstring-stderr=%s", text, text))
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err = c.Start()
	assert.NoError(err)
	err = c.Wait()
	assert.NoError(err)

	out := restoreOutput()

	assert.Contains(out, "randstring-println="+text)
	assert.Contains(out, "randstring-stdout="+text)
	assert.Contains(out, "randstring-stderr="+text)
}

// TestConfirmTo ensures that the confirmation prompt works as expected.
func TestConfirmTo(t *testing.T) {
	assert := asrt.New(t)

	// Unset the env var, then reset after the test
	t.Setenv("DDEV_NONINTERACTIVE", "")

	// test a given input against a default value
	testInput := func(input string, defaultTo bool, expected bool) {
		text := util.RandString(32)
		scanner := bufio.NewScanner(strings.NewReader(input))
		util.SetInputScanner(scanner)

		getOutput := util.CaptureStdOut()
		resp := util.ConfirmTo(text, defaultTo)
		if expected {
			assert.True(resp)
		} else {
			assert.False(resp)
		}
		assert.Contains(getOutput(), text)
	}

	yesses := []string{"YES", "Yes", "yes", "Y", "y"}
	for _, y := range yesses {
		testInput(y, true, true)
		testInput(y, false, true)
	}

	nos := []string{"NO", "No", "no", "N", "n"}
	for _, n := range nos {
		testInput(n, true, false)
		testInput(n, false, false)
	}

	// Test that junk answers (not yes, no, or <enter>) eventually return a false
	junkText := "a\nb\na\nb\na\nb\na\nb\na\nb\n"
	testInput(junkText, true, false)
	testInput(junkText, false, false)

	// Test that <enter> returns the defaultTo value
	enter := "\n"
	testInput(enter, true, true)
	testInput(enter, false, false)
}

// TestSliceToUniqueSlice tests SliceToUniqueSlice
func TestSliceToUniqueSlice(t *testing.T) {
	assert := asrt.New(t)

	testBedSources := [][]string{
		{"1", "2", "3", "2", "3", "1"},
		{"99", "98", "97", "99", "98", "97", "1", "2", "3"},
	}

	testBedExpectations := [][]string{
		{"1", "2", "3"},
		{"99", "98", "97", "1", "2", "3"},
	}

	for i := 0; i < len(testBedSources); i++ {
		res := util.SliceToUniqueSlice(&testBedSources[i])
		assert.Equal(testBedExpectations[i], res)
	}
}

// TestArrayToReadableOutput tests ArrayToReadableOutput
func TestArrayToReadableOutput(t *testing.T) {
	assert := asrt.New(t)

	testSource := []string{
		"file1.conf",
		"file2.conf",
	}
	expectation := `
  - file1.conf
  - file2.conf
`
	res, _ := util.ArrayToReadableOutput(testSource)
	assert.Equal(expectation, res)

	_, err := util.ArrayToReadableOutput([]string{})
	assert.EqualErrorf(err, "empty slice", "Expected error when passing an empty slice")
}

// TestGetTimezone tests GetTimezone
func TestGetTimezone(t *testing.T) {
	testCases := []struct {
		description string
		input       string
		result      string
		error       string
	}{
		{"$TZ env var", "Europe/London", "Europe/London", ""},
		{"Linux /etc/localtime symlink", "/usr/share/zoneinfo/Europe/London", "Europe/London", ""},
		{"Linux /etc/localtime posix symlink", "/usr/share/zoneinfo/posix/Europe/London", "Europe/London", ""},
		{"Linux /etc/localtime right symlink", "/usr/share/zoneinfo/right/Europe/London", "Europe/London", ""},
		{"/etc/localtime is not a symlink", "/etc/localtime", "", "unable to read timezone from /etc/localtime"},
		{"macOS /etc/localtime symlink", "/var/db/timezone/zoneinfo/Europe/London", "Europe/London", ""},
		{"macOS Sonoma /etc/localtime symlink", "/private/var/db/timezone/tz/2024a.1.0/zoneinfo/Europe/London", "Europe/London", ""},
		{"macOS Sequoia /etc/localtime symlink", "/usr/share/zoneinfo.default/Europe/London", "Europe/London", ""},
		{"Case-insensitive search for /zoneinfo/ in the path", "/path/to/TestZoneInfoTest/Europe/London", "Europe/London", ""},
		{"Timezone has wrong format", "/Europe/London", "", "unable to read timezone from /Europe/London"},
		{"Not real timezone", "Europe/Test", "", "unable to read timezone from Europe/Test"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			timezone, err := util.GetTimezone(tc.input)
			require.Equal(t, tc.result, timezone)
			if tc.error == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.error)
			}
		})
	}
}

// compareSlices checks if two slices are equal after sorting them.
func compareSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Sort both slices before comparing
	sort.Strings(a)
	sort.Strings(b)

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestWarningOnce tests that WarningOnce only shows each unique message once
func TestWarningOnce(t *testing.T) {
	assert := asrt.New(t)

	// Capture output to test warning messages (warnings go to UserErr)
	restoreOutput := util.CaptureUserErr()

	// Test that same message is only shown once
	util.WarningOnce("test warning message")
	util.WarningOnce("test warning message")
	util.WarningOnce("test warning message")

	// Test that different messages are both shown
	util.WarningOnce("different warning message")

	out := restoreOutput()

	// Should contain each unique message only once
	assert.Equal(1, strings.Count(out, "test warning message"))
	assert.Equal(1, strings.Count(out, "different warning message"))

	// Test with format arguments
	restoreOutput = util.CaptureUserErr()

	util.WarningOnce("warning with %s", "arg1")
	util.WarningOnce("warning with %s", "arg1")
	util.WarningOnce("warning with %s", "arg2")

	out = restoreOutput()

	// Should show each formatted message once
	assert.Equal(1, strings.Count(out, "warning with arg1"))
	assert.Equal(1, strings.Count(out, "warning with arg2"))
}

// TestSubtractSlices does simple test of SubtractSlices
func TestSubtractSlices(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{
			name:     "No overlap",
			a:        []string{"apple", "banana", "cherry"},
			b:        []string{"date", "fig"},
			expected: []string{"apple", "banana", "cherry"},
		},
		{
			name:     "Partial overlap",
			a:        []string{"apple", "banana", "cherry"},
			b:        []string{"banana"},
			expected: []string{"apple", "cherry"},
		},
		{
			name:     "Complete overlap",
			a:        []string{"apple", "banana"},
			b:        []string{"apple", "banana"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.SubtractSlices(tt.a, tt.b)
			if !compareSlices(result, tt.expected) {
				t.Errorf("SubtractSlices(%v, %v) = %v; want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestGetHomeDir tests GetHomeDir
func TestGetHomeDir(t *testing.T) {
	home := util.GetHomeDir()
	if home == "" {
		t.Fatal("returned home directory is empty")
	}

	if !filepath.IsAbs(home) {
		t.Fatalf("returned path is not absolute: %s", home)
	}
}

// TestFindBashPath tests FindBashPath
func TestFindBashPath(t *testing.T) {
	assert := asrt.New(t)

	bashPath := util.FindBashPath()

	// On non-Windows systems, it should return "bash"
	if !nodeps.IsWindows() {
		assert.Equal("bash", bashPath)
		return
	}

	// On Windows, if bash is found, it should return a valid path
	// We can't guarantee a specific path because it depends on the installation
	if bashPath != "" {
		// Should be an absolute path
		require.True(t, filepath.IsAbs(bashPath), "Expected absolute path, got: %s", bashPath)

		// Should end with bash.exe on Windows
		assert.True(strings.HasSuffix(strings.ToLower(bashPath), "bash.exe"),
			"Expected path ending with bash.exe, got: %s", bashPath)

		// Verify the file exists
		_, err := os.Stat(bashPath)
		require.NoError(t, err, "Bash path %s does not exist", bashPath)
	}
	// Note: We don't fail if bashPath is empty because bash might not be installed
	// in the test environment
}

// TestFormatBytes tests the byte formatting function
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"Zero bytes", 0, "0B"},
		{"Small bytes", 512, "512B"},
		{"Exactly 1KB", 1024, "1.0KB"},
		{"Multiple KB", 2560, "2.5KB"},
		{"Exactly 1MB", 1024 * 1024, "1.0MB"},
		{"Fractional MB", 1024*1024 + 512*1024, "1.5MB"},
		{"Large MB", 157286400, "150.0MB"},
		{"Exactly 1GB", 1024 * 1024 * 1024, "1.0GB"},
		{"Fractional GB", int64(1024*1024*1024) + int64(512*1024*1024), "1.5GB"},
		{"Multiple GB", int64(5) * 1024 * 1024 * 1024, "5.0GB"},
		{"Exactly 1TB", int64(1024) * 1024 * 1024 * 1024, "1.0TB"},
		{"Large volume", 982700000, "937.2MB"}, // Approximate d11 volume size
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.FormatBytes(tt.bytes)
			require.Equal(t, tt.expected, result, "FormatBytes(%d) should return %s, got %s", tt.bytes, tt.expected, result)
		})
	}
}

// TestExtractCurlBody demonstrates extracting response body from curl output with diagnostics.
//
// This pattern is useful when you need both:
// - Parseable response body (e.g., JSON for validation)
// - HTTP status code visible in error messages for debugging
//
// Usage:
//
//	out, err := exec.RunHostCommand("curl", "-sf",
//	    "-w", util.CurlDiagnosticSuffix+"%{http_code}",
//	    "http://example.com/api/endpoint")
//	require.NoError(t, err, "curl failed, output='%s'", out)  // Full output with HTTP code for debugging
//	body := util.ExtractCurlBody(out)                          // Clean body for JSON parsing
//	var result map[string]interface{}
//	json.Unmarshal([]byte(body), &result)
func TestExtractCurlBody(t *testing.T) {
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl not available, skipping test")
	}

	// Run actual curl command with -w to append HTTP code
	cmd := exec.Command("curl", "-sf",
		"-w", util.CurlDiagnosticSuffix+"%{http_code}",
		"https://httpbin.org/json")
	outBytes, err := cmd.CombinedOutput()
	out := string(outBytes)

	require.NoError(t, err, "curl command failed, output='%s'", out)

	// The raw output must include the HTTP code suffix we added
	require.Contains(t, out, "_CURL_HTTP_CODE_:200",
		"raw output should contain HTTP code suffix, got: %s", out)

	// ExtractCurlBody strips the diagnostic suffix, leaving clean JSON
	body := util.ExtractCurlBody(out)

	// Verify extraction actually removed something
	require.Less(t, len(body), len(out),
		"extracted body should be shorter than raw output (suffix removed)")

	// Verify the suffix is gone from extracted body
	require.NotContains(t, body, "_CURL_HTTP_CODE_",
		"extracted body should not contain diagnostic suffix, got: %s", body)

	// Verify the body is valid JSON that can be parsed
	require.True(t, strings.HasPrefix(strings.TrimSpace(body), "{"),
		"extracted body should be valid JSON, got: %s", body)
}
