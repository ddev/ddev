<h1>Drud-S3 Hosting Provider Integration</h1>

ddev provides an integration with the [Drud-S3 Website Management Platform (DDEV-Live)](https://www.drud.com/ddev-live/), which allows for ddev-live users to quickly download and provision a project via their Amazon AWS S3 backups bucket in a local ddev-managed environment.

ddev's Drud-S3 integration pulls an existing backup from an existing AWS S3 backups bucket into your local system so you can develop locally. Of course that means you must already have a DDEV-Live site with a backup in order to use it.

## Quick Start

If you have ddev installed, and have an active DDEV-Live account, you can follow this quick start guide to spin up a local project.

1. Choose a Drud-S3 site and environment you want to use with ddev.

2. Get a copy of the project codebase from DDEV-Live via the git credentials provided.

3. Configure the local checkout for ddev.

    a. Navigate in your terminal to your checkout of the project codebase.

    b. Run `ddev config drud-s3`. When asked for the project name you must use the exact name of the Drud-S3 project.

    c. Configuration prompts will allow you to provide the AWS S3 credentials, bucket name, and environment.

4. Run `ddev pull`. The ddev environment will spin up, download the latest backup, and import the database and files into the ddev environment. You should now be able to access the project locally.

_**Note for WordPress Users:** In order for your local project to load file assets from your local environment rather than the Drud-S3 environment it was pulled from, the URL of the project must be changed in the database by performing a search and replace. ddev provides an example `wp search-replace` as a post-pull hook in the config.yaml for your project. It is recommended to populate and uncomment this example so that replacement is done any time a backup is pulled from Drud-S3._
