# Adding New Project Types

Adding and maintaining project types (like `typo3`, `magento2`, etc.) is not too hard. Please update and add to this doc when you find things that have been missed.

To add a new project type:
* Add the new type to the list in `nodeps.go`
* Add to `appTypeMatrix` in `apptypes.go`
* Create a new go file for your project type, like `django.go`.
* Implement the functions that you think are needed for your project type and add references to them in your `appTypeMatrix` stanza. There are lots of examples that you can start with in places like `drupal.go` and `typo3.go`, `shopware6.go`, etc. The comments in the code in `apptypes.go` for the `appTypeFuncs` for each type of action tell what these are for, but here's a quick summary.
  * `settingsCreator` is the function that will create a main settings file if none exists. 
  * `uploadDir` returns the filepath of the user-uploaded files directory for the project type, like `sites/default/files` for Drupal or `pub/media` for magento2.
  * `hookDefaultComments` adds comments to config.yaml about hooks with an example for that project type. It's probably not useful at all.
  * `apptypeSettingsPaths` returns the paths for the main settings file and the extra settings file that ddev may create (like settings.ddev.php for Drupal).
  * `appTypeDetect` is a function that determines whether the project is of the type you're implementing.
  * `postImportDBAction` can do something after db import. I don't see it implemented anywhere.
  * `configOverrideAction` can change default config for your project type. For example, magento2 now requires `php8.1`, so a `configOverrideAction` can change the php version.
  * `postConfigAction` gives a chance to do something at the end of config... but it doesn't seem to be used anywhere.
  * `postStartAction` adds actions at the end of `ddev start`. You'll see several implementations of this, for things like creating needed default directories, or setting permissions on files, etc.
  * `importFilesAction` defines how `ddev import-files` works for this project type.
  * `defaultWorkingDirMap` allows the project type to override the project's `working_dir` (where `ddev ssh` and `ddev exec` start by default). This is mostly not done any more, as the `working_dir` is typically the project root.
* You'll likely need templates for settings files, use the drupal or typo3 templates as examples, for example `pkg/ddevapp/drupal` and `pkg/ddevapp/typo3`. Those templates have to be loaded at runtime as well.
* Once your project type starts working and behaving the way you want it to, you'll need to add test artifacts for it and try testing it (locally first). 
  * Add your project to `TestSites` in ddevapp_test.go.
  * Create a DDEV project named `testpkg<projectype>` somewhere and get it going and working with a database and files you can export.
  * Export the database, files, and (optionally) code to tarballs or `.sql.gz`. Put them somewhere on the internet- they'll end up in `drud/ddev_test_tarballs` - I can give you permissions on that if you like. The magento2 project has descriptions explaining how each tarball gets created. Do that for yours as well.
  * Run the test and get it working. I usually use the trick of setting `GOTEST_SHORT=<element_in_TestSites>`, like `GOTEST_SHORT=7`. Then set that environment variable in the Goland profile or your environment. `export GOTEST_SHORT=7 && make testpkg TEST_ARGS="-run TestDdevFullsiteSetup"`
