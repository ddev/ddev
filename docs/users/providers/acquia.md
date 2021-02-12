## Acquia Hosting Provider Integration

ddev provides integration with the [Acquia Cloud Platform](https://www.acquia.com/choosing-right-acquia-cloud-platform), which allows Acquia users to quickly download and provision a project from Acquia in a local ddev-managed environment.

ddev's Acquia integration pulls database and files from an existing project into your local system so you can develop locally.

### Acquia Quickstart

If you have ddev installed, and have an active Acquia account with an active site, you can follow this guide to spin up a acquia project locally.

1. Get your Acquia API token from the Account Settings->API Tokens.
2. Make sure your ssh key is authorized on your Acquia account at Account Settings->SSH Keys
3. `ddev auth ssh` (this typically needs only be done once per ddev session, not every pull.)
4. Add add an entry to web_environment in ~/.ddev/global_config.yaml with the token:

   ```yaml
   web_environment:
   - ACQUIA_API_TOKEN=xxxxxxxx
   - ACQUIA_API_SECRET=xxxxx
   ```

5. Copy acquia.yaml.example to acquia.yaml.
6. Update project_id.
7. Your project must include drush; `ddev composer require drush/drush` if it isn't there already.
8. `ddev restart`
9. Use `ddev pull acquia` to pull the project database and files.
