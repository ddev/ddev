## CMS-Specific Help and Techniques

### Drupal Specifics

#### Settings files

By default, DDEV will create settings files for your project that make it "just work" out of the box. It creates a `sites/default/settings.ddev.php` and adds an include in `sites/default/settings.php` to bring that in. There are guards to prevent the `settings.ddev.php` from being active when the project is not running under DDEV, but it still should not be checked in and is gitignored.

### TYPO3 Specifics

#### Settings files

### Running any PHP App with DDEV
