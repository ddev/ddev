## CMS-Specific Help and Techniques

### Drupal Specifics

* **Settings Files**: By default, DDEV will create settings files for your project that make it "just work" out of the box. It creates a `sites/default/settings.ddev.php` and adds an include in `sites/default/settings.php` to bring that in. There are guards to prevent the `settings.ddev.php` from being active when the project is not running under DDEV, but it still should not be checked in and is gitignored.

### TYPO3 Specifics

* **Settings Files**: On `ddev start`, DDEV creates a `public/typo3conf/AdditionalConfiguration.php` with database configuration in it.

### Running any PHP App with DDEV

Nearly any PHP app will run fine with DDEV, and lots of others. If your project type is not one of the explicitly supported project types, that's fine. Just set the project type to 'php' and go about setting up settings files or .env as you normally would.
