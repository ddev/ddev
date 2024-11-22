---
search:
  boost: .5
---
# Adding New Project Types

Adding and maintaining project types (like `typo3`, `magento2`, etc.) is not too hard. Please update and add to this doc when you find things that have been missed.

To add a new project type:

* Add the new type to the list in `nodeps/values.go`
* Add to `appTypeMatrix` in `apptypes.go`
* Add to `properties.type` in `ddevapp/schema.json`
* Add to `project_type` test loop in `ddev-webserver/test.sh`
* Add to `type` in `templates.go` comments
* Create a new go file for your project type, like `mytype.go`.
* Implement the functions that you think are needed for your project type and add references to them in your `appTypeMatrix` stanza. There are lots of examples that you can start with in places like `drupal.go` and `typo3.go`, `shopware6.go`, etc. The comments in the code in `apptypes.go` for the `appTypeFuncs` for each type of action tell what these are for, but here's a quick summary.
    * `settingsCreator` is the function that will create a main settings file if none exists.
    * `uploadDir` returns the filepath of the user-uploaded files directory for the project type, like `sites/default/files` for Drupal or `pub/media` for magento2.
    * `hookDefaultComments` adds comments to `config.yaml` about hooks with an example for that project type. It's probably not useful at all.
    * `apptypeSettingsPaths` returns the paths for the main settings file and the extra settings file that DDEV may create (like `settings.ddev.php` for Drupal).
    * `appTypeDetect` is a function that determines whether the project is of the type you’re implementing.
    * `postImportDBAction` can do something after db import. I don’t see it implemented anywhere.
    * `configOverrideAction` can change default config for your project type. For example, your CMS may require `php8.3`, so a `configOverrideAction` can change the php version.
    * `postConfigAction` gives a chance to do something at the end of config, but it doesn’t seem to be used anywhere.
    * `postStartAction` adds actions at the end of [`ddev start`](../users/usage/commands.md#start). You'll see several implementations of this, for things like creating needed default directories, or setting permissions on files, etc.
    * `importFilesAction` defines how [`ddev import-files`](../users/usage/commands.md#import-files) works for this project type.
    * `defaultWorkingDirMap` allows the project type to override the project’s [`working_dir`](../users/configuration/config.md#working_dir) (where [`ddev ssh`](../users/usage/commands.md#ssh) and [`ddev exec`](../users/usage/commands.md#exec) start by default). This is mostly not done anymore, as the `working_dir` is typically the project root.
    * `composerCreateAllowedPaths` specifies the paths that can exist in a directory when `ddev composer create` is being used.
* You’ll likely need templates for settings files, use the Drupal or TYPO3 templates as examples, for example `pkg/ddevapp/drupal` and `pkg/ddevapp/typo3`. Those templates have to be loaded at runtime as well.
* For a custom nginx config, use `webserver_config_assets/nginx-site-php.conf` as an example.
* If the project type has a custom command, add it to `global_dotddev_assets/commands/web` folder.
* Once your project type starts working and behaving as you’d like, you’ll need to add test artifacts for it and try testing it (locally first).
    * Add your project to `TestSites` in `ddevapp_test.go`.
    * Create a DDEV project named `testpkg<projectype>` somewhere and get it going and working with a database and files you can export.
    * Export the database, files, and (optionally) code to tarballs or `.sql.gz`. Put them somewhere on the internet—they’ll end up in `ddev/test-<projectype>`. We will give you permissions on that if you like. The `magento2` project has descriptions explaining how each tarball gets created. Do that for yours as well.
    * Run the test and get it working. I usually use the trick of setting `GOTEST_SHORT=<element_in_TestSites>`, like `GOTEST_SHORT=7`. Then set that environment variable in the GoLand profile or your environment. `export GOTEST_SHORT=7 && make testpkg TEST_ARGS="-run TestDdevFullsiteSetup"`
* (We can assist you with this step if needed) Upload your project to GitHub or create an upstream fork ([example for Laravel](https://github.com/ddev/test-laravel)):
    * Create a `ddev-automated-test` branch, set it as the default.
    * Commit all vendor dependencies, but don't commit the `.ddev` directory.
    * Update the `README.md` instructions.
    * Create a new release and attach the database and file artifacts.
    * Transfer the repository to DDEV, we will maintain your access. (Ask us to do this after opening the PR.)
* Update the documentation:
    * If it doesn't pass our spellcheck, add the word to `.spellcheckwordlist.txt`
    * Add the new type to `users/configuration/config.md`
    * Update the `users/quickstart.md`
    * If there is a new command, add it to `users/usage/commands.md` and `users/usage/cli.md`
