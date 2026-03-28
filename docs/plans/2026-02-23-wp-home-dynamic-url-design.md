# Dynamic WP_HOME via Runtime Environment Variables - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace hardcoded WP_HOME URL in wp-config-ddev.php with runtime `getenv()` calls so WordPress dynamically adapts to tunnel/share URLs.

**Architecture:** Single template file change in `pkg/ddevapp/wordpress/wp-config-ddev.php`. Replace `{{ $config.DeployURL }}` with `getenv('DDEV_SHARE_URL') ?: getenv('DDEV_PRIMARY_URL')`. No Go code changes needed. Test update to verify new template output.

**Tech Stack:** PHP template (Go text/template), Go tests

**Issue:** [#8098](https://github.com/ddev/ddev/issues/8098)

---

### Task 1: Update wp-config-ddev.php Template

**Files:**
- Modify: `pkg/ddevapp/wordpress/wp-config-ddev.php:22-26`

**Step 1: Edit the template**

Replace lines 22-26 in `pkg/ddevapp/wordpress/wp-config-ddev.php`:

Old:
```php
	/** WP_HOME URL */
	defined( 'WP_HOME' ) || define( 'WP_HOME', '{{ $config.DeployURL }}' );

	/** WP_SITEURL location */
	defined( 'WP_SITEURL' ) || define( 'WP_SITEURL', WP_HOME . '/{{ $config.AbsPath  }}' );
```

New:
```php
	/** WP_HOME URL */
	$wp_home = getenv('DDEV_SHARE_URL') ?: getenv('DDEV_PRIMARY_URL');
	defined( 'WP_HOME' ) || define( 'WP_HOME', $wp_home );

	/** WP_SITEURL location */
	defined( 'WP_SITEURL' ) || define( 'WP_SITEURL', WP_HOME . '/{{ $config.AbsPath  }}' );
```

**Step 2: Verify template is valid**

Run: `go vet ./pkg/ddevapp/...`
Expected: No errors (template is embedded, `go vet` catches basic issues)

**Step 3: Commit**

```bash
git add pkg/ddevapp/wordpress/wp-config-ddev.php
git commit -m "feat(wordpress): use runtime env vars for WP_HOME instead of hardcoded URL, fixes #8098"
```

---

### Task 2: Update Test to Verify New Template Output

**Files:**
- Modify: `pkg/ddevapp/settings_test.go:70-85`

The existing `TestWriteSettings` test (line 34) verifies WordPress generates `wp-config-ddev.php` with the DDEV signature. We need to add a check that the generated file contains `getenv('DDEV_PRIMARY_URL')` instead of a literal URL.

**Step 1: Write the test assertion**

Add after line 83 in `pkg/ddevapp/settings_test.go`, inside the `for apptype` loop, after the signature check:

```go
		// For WordPress, verify WP_HOME uses runtime env vars instead of hardcoded URL
		if apptype == nodeps.AppTypeWordPress {
			envVarFound, err := fileutil.FgrepStringInFile(expectedSettingsFile, "getenv('DDEV_PRIMARY_URL')")
			assert.NoError(err)
			assert.True(envVarFound, "Expected getenv('DDEV_PRIMARY_URL') in %s for dynamic WP_HOME", expectedSettingsFile)

			hardcodedURLFound, err := fileutil.FgrepStringInFile(expectedSettingsFile, "DeployURL")
			assert.NoError(err)
			assert.False(hardcodedURLFound, "Should not find hardcoded DeployURL template in generated %s", expectedSettingsFile)
		}
```

**Step 2: Run the test to verify it passes**

Run: `go test -v -run TestWriteSettings ./pkg/ddevapp/`
Expected: PASS - the generated file should contain `getenv('DDEV_PRIMARY_URL')` and not contain a `DeployURL` template reference.

**Step 3: Commit**

```bash
git add pkg/ddevapp/settings_test.go
git commit -m "test(wordpress): verify WP_HOME uses runtime env vars"
```

---

### Task 3: Run Static Analysis

**Step 1: Run staticrequired**

Run: `make staticrequired`
Expected: PASS with no lint errors

**Step 2: Fix any issues**

If gofmt or linting issues arise, fix and re-run.

---

### Task 4: Final Verification

**Step 1: Build**

Run: `make`
Expected: Successful build

**Step 2: Run broader test suite for WordPress-related packages**

Run: `go test -v -run "TestWriteSettings|TestWordpress" ./pkg/ddevapp/`
Expected: All tests pass

**Step 3: Verify generated file manually**

Run: `go test -v -run TestWriteSettings ./pkg/ddevapp/ 2>&1 | head -20`
Check output shows PASS.
