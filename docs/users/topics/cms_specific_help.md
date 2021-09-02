## CMS-Specific Help and Techniques

### Drupal Specifics

* **Settings Files**: By default, DDEV will create settings files for your project that make it "just work" out of the box. It creates a `sites/default/settings.ddev.php` and adds an include in `sites/default/settings.php` to bring that in. There are guards to prevent the `settings.ddev.php` from being active when the project is not running under DDEV, but it still should not be checked in and is gitignored.

### TYPO3 Specifics

* **Settings Files**: On `ddev start`, DDEV creates a `public/typo3conf/AdditionalConfiguration.php` with database configuration in it.

#### Setup a Base Variant (since TYPO3 9.5)

Since TYPO3 9.5 you have to setup a `Site Configuration` for each site you like to serve. To be able to browse the site on your local environment you have to setup a `Base Variant` in your `Site Configuration` depending on your local context. In this example we assume a `Application Context` `Development/DDEV` which can be set in the DDEV's `config.yaml`:

```yaml
web_environment:
- TYPO3_CONTEXT=Development/DDEV
```

This variable will be available after the project start or restart.

Afterwards add a `Base Variant` to your `Site Configuration`:

```yaml
baseVariants:
  -
    base: 'https://example.com.ddev.site/'
    condition: 'applicationContext == "Development/DDEV"'
```

See also [TYPO3 Documentation](https://docs.typo3.org/m/typo3/reference-coreapi/master/en-us/ApiOverview/SiteHandling/BaseVariants.html).

### Running any PHP App with DDEV

Nearly any PHP app will run fine with DDEV, and lots of others. If your project type is not one of the explicitly supported project types, that's fine. Just set the project type to 'php' and go about setting up settings files or .env as you normally would.
