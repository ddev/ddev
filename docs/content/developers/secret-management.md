# Secret Management

Most secrets used in our build process are managed via 1Password. The technique is documented in [developer.1password.com](https://developer.1password.com/docs/ci-cd/github-actions/).

## Pull and Push Secrets

Secrets for the TestPlatformPull, TestAcquiaPull, and similar tests are in the `test-secrets` vault in the DDEV 1Password instance. They can be rotated and otherwise managed there.

The `test-secrets` vault allows access to the service account [`tests`](https://team-ddev.1password.com/developer-tools/infrastructure-secrets/serviceaccount/76URGQSBMVEUHPEWVHQRFJ4N3Q) whose auth token is in "1Password Service Account Auth Token: Tests"
