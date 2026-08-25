---
search:
  boost: 3
---

# Environment Variables

There are three different things people mean by "environment variables" in a DDEV project, and they land in different places. This page covers all of them, in what order they override each other, and where to put a secret.

The same two scopes as [config options](config.md) apply here:

1. :octicons-file-directory-16: **Project** files, in the project directory and mostly under `.ddev`.
2. :octicons-globe-16: **Global** files in the [global DDEV directory](../usage/architecture.md#global-files), which apply to every project. This page writes that directory as `$HOME/.ddev`, its usual location, but it can be somewhere else.

!!!tip "Container variables take effect on restart"
    The `.ddev/.env*` files, their global equivalents, and `web_environment` are read while containers are being created, so run [`ddev restart`](../usage/commands.md#restart) after changing them. An application `.env` file in the project root is not one of these: your framework reads it on every request, so changing it needs no restart.

## Which Variables Reach Where

| Where you set it | Type | What it reaches | Who reads it |
| -- | -- | -- | -- |
| An application `.env` file in the project root | :octicons-file-directory-16: project | Nothing in the container environment; it stays a file in your project | Your framework: Laravel, Craft CMS, Symfony, Silverstripe, and others |
| [`.ddev/.env*`](#project-env-files) and [`$HOME/.ddev/.env*`](#global-env-files) | :octicons-file-directory-16: project<br>:octicons-globe-16: global | The environment of any container in the project: `web`, `db`, and add-on or custom services | Anything running in the container, plus interpolation in `.ddev/docker-compose.*.yaml` |
| [`web_environment`](#web_environment) | :octicons-file-directory-16: project<br>:octicons-globe-16: global | The environment of the `web` container only | Anything running in `web` |

## Application `.env` Files, Outside `.ddev/`

Many frameworks read a `.env` file in the project root. That file belongs to your application, and DDEV treats it as ordinary project code: nothing in it becomes a container environment variable.

For several project types DDEV writes database credentials and the project URL into that file as part of [settings management](../usage/cms-settings.md), creating it if it doesn’t exist. Set [`disable_settings_management`](config.md#disable_settings_management) to `true` if you want to manage the file yourself.

You can edit it from the command line with [`ddev dotenv set`](../usage/commands.md#dotenv-set), which is useful in a `README` or a CI script because it creates the file and updates single keys without disturbing the rest:

```bash
ddev dotenv set .env --app-key=value
```

Your framework reads this file on every request, so a change to it takes effect without a `ddev restart`.

The `.local` convention below applies only inside `.ddev/` and `$HOME/.ddev/`. A root `.env.local`, like the one Symfony and Shopware use, is your project’s file and is covered by your project’s own `.gitignore`.

## Project Env Files

The `.env*` files in the project’s `.ddev` directory set real environment variables in the project’s containers. The file name decides which service gets them:

```text
.env[.<service>[.<label>...]][.local]
```

| File | Reaches | Notes |
| -- | -- | -- |
| `.ddev/.env` | Every container in the project | |
| `.ddev/.env.local` | Every container in the project | Gitignored, DDEV v1.25.4+ |
| `.ddev/.env.web` | The `web` container | Any service name works: `db`, `redis`, and so on |
| `.ddev/.env.web.local` | The `web` container | Gitignored, DDEV v1.25.4+ |
| `.ddev/.env.web.myaddon` | The `web` container | A label keeps files from different sources apart, DDEV v1.25.4+ |
| `.ddev/.env.web.myaddon.local` | The `web` container | Gitignored, DDEV v1.25.4+ |
| `.ddev/.env.redis-build` | Nothing | No service is named `redis-build`, so this file is expanded but never injected |

A trailing `.local` only marks the file as gitignored, and a label after the service name only keeps files apart. Both are dropped before DDEV reads the service name, so `local` cannot be used as a service or label name. Files ending in `.example` are skipped, which is how an add-on ships a documented template without setting anything.

Every one of these files is also passed to `docker-compose config`, whatever it is named, so `${SOME_VARIABLE}` in a `.ddev/docker-compose.*.yaml` file is expanded from all of them together. That is the point of the `.ddev/.env.redis-build` row above: when a variable is only meant to be substituted into a compose file, give the file a name that is not a service name and it will never appear inside a container.

## Override Order

Later wins. Everything in the [global DDEV directory](../usage/architecture.md#global-files) is applied before anything belonging to the project, and within each directory DDEV sorts the files by their name with a trailing `.local` removed, which puts a `.local` file directly after the file it overrides.

The table below is a project that has every kind of file at once, with `db` and `web` as its services and `myaddon` as a label. The global files, the `.local` files, and the labeled files all need DDEV v1.25.4+; `.ddev/.env` and `.ddev/.env.<service>` are older.

| # | Content | Notes |
| -- | -- | -- |
| 1 | :octicons-globe-16: `web_environment` in `$HOME/.ddev/global_config.yaml` | Reaches the `web` container only |
| 2 | :octicons-file-directory-16: `web_environment` in `.ddev/config.yaml`, then `.ddev/config.*.yaml` | Reaches the `web` container only |
| 3 | :octicons-globe-16: `$HOME/.ddev/.env` | Every container of every project |
| 4 | :octicons-globe-16: `$HOME/.ddev/.env.local` | Overrides the file above it, gitignored |
| 5 | :octicons-globe-16: `$HOME/.ddev/.env.db` | Services in alphabetical order, so `db` before `web` |
| 6 | :octicons-globe-16: `$HOME/.ddev/.env.db.local` | Overrides the file above it, gitignored |
| 7 | :octicons-globe-16: `$HOME/.ddev/.env.web` | &zwnj; |
| 8 | :octicons-globe-16: `$HOME/.ddev/.env.web.local` | Overrides the file above it, gitignored |
| 9 | :octicons-globe-16: `$HOME/.ddev/.env.web.myaddon` | Labels in alphabetical order, all after the unlabeled `$HOME/.ddev/.env.web` |
| 10 | :octicons-globe-16: `$HOME/.ddev/.env.web.myaddon.local` | Overrides the file above it, gitignored |
| 11 | :octicons-file-directory-16: `.ddev/.env` | Every container of this project |
| 12 | :octicons-file-directory-16: `.ddev/.env.local` | Overrides the file above it, gitignored |
| 13 | :octicons-file-directory-16: `.ddev/.env.db` | Services in alphabetical order, so `db` before `web` |
| 14 | :octicons-file-directory-16: `.ddev/.env.db.local` | Overrides the file above it, gitignored |
| 15 | :octicons-file-directory-16: `.ddev/.env.web` | &zwnj; |
| 16 | :octicons-file-directory-16: `.ddev/.env.web.local` | Overrides the file above it, gitignored |
| 17 | :octicons-file-directory-16: `.ddev/.env.web.myaddon` | Labels in alphabetical order, all after the unlabeled `.ddev/.env.web` |
| 18 | :octicons-file-directory-16: `.ddev/.env.web.myaddon.local` | Overrides the file above it, gitignored, and applied last of all |

Only the files you actually have are applied, and the rest of the order is unchanged. A second label, `.ddev/.env.web.otheraddon`, would come after row 18, because `myaddon` sorts before `otheraddon` and a `.local` file never leaves the file it overrides.

To see the result:

* [`ddev utility check-custom-config`](../usage/commands.md#utility-check-custom-config) lists the env files DDEV found under `Environment`, in the order they are applied.
* [`ddev utility compose-config`](../usage/commands.md#utility-compose-config) shows the rendered `.ddev/.ddev-docker-compose-full.yaml`, including the `environment` section of each service.
* `ddev exec env` and `ddev exec -s <service> env` show what actually reached a running container.

## Global Env Files

With DDEV v1.25.4+, the same file names work in the [global configuration directory](../usage/architecture.md#global-files), where they apply to every project on the machine. `$HOME/.ddev/.env` reaches every container of every project, and `$HOME/.ddev/.env.db` reaches the `db` container of every project.

This is the only way to set a variable globally for something other than the `web` container, which is all [`web_environment`](#web_environment) can do.

Use [`ddev dotenv global set`](../usage/commands.md#dotenv-global-set) to write them:

```bash
ddev dotenv global set .env.web --api-url=https://example.com
```

## Secrets

With DDEV v1.25.4+, DDEV adds `/.env.local` and `/.env.*.local` to both `.ddev/.gitignore` and `$HOME/.ddev/.gitignore`, so any env file ending in `.local` stays out of Git without you having to take over those files.

That gives a shared service a natural split: commit `.ddev/.env.web` with the settings your team shares, and keep the token next to it in `.ddev/.env.web.local`, which is applied right after it.

```bash
# Committed, everyone gets it
ddev dotenv set .ddev/.env.web --api-url=https://example.com

# Gitignored, yours only
ddev dotenv set .ddev/.env.web.local --api-key=secret
```

To document which keys are expected without committing their values, add a `.ddev/.env.web.example` file listing the keys with empty values. DDEV never reads `.example` files, and `.ddev/.gitignore` ignores them, so commit it with `git add -f .ddev/.env.web.example`.

The global files work the same way, so `$HOME/.ddev/.env.web.local` is a reasonable home for a credential you use in every project.

## `web_environment`

The [`web_environment`](config.md#web_environment) setting predates the env files above and only reaches the `web` container. It lives in `.ddev/config.yaml` or any `.ddev/config.*.yaml` for a project, and in `$HOME/.ddev/global_config.yaml` for every project:

```yaml
web_environment:
    - MY_ENV_VAR=someval
    - MY_OTHER_ENV_VAR=someotherval
```

You can also pass a variable through from the host by listing just its name with no value. DDEV hands the bare name to docker-compose, which resolves it from the host environment when the container starts:

```yaml
web_environment:
    - MY_HOST_VAR
```

To set these from the command line, use [`ddev config`](../usage/commands.md#config). Use `--web-environment` instead of `--web-environment-add` to replace the existing list rather than add to it:

```bash
# Set MY_ENV_VAR for the project
ddev config --web-environment-add="MY_ENV_VAR=someval"

# Set MY_ENV_VAR globally
ddev config global --web-environment-add="MY_ENV_VAR=someval"
```

## Applying Changes

A running container never picks up a new value on its own. After adding or editing any `.ddev/.env*` file, its global equivalent, or a `web_environment` value, run:

```bash
ddev restart
```

An application `.env` file in the project root is the exception, since your framework reads it on every request rather than at container start.

## `ddev dotenv`

The [`ddev dotenv`](../usage/commands.md#dotenv) commands create and update these files without an editor, quoting values correctly. Flags become variable names: `--api-url` writes `API_URL`.

| Command | Type | Path is relative to |
| -- | -- | -- |
| [`ddev dotenv set`](../usage/commands.md#dotenv-set), [`ddev dotenv get`](../usage/commands.md#dotenv-get) | :octicons-file-directory-16: project | The project root, so `.env` for an application file and `.ddev/.env.web` for a DDEV one |
| [`ddev dotenv global set`](../usage/commands.md#dotenv-global-set), [`ddev dotenv global get`](../usage/commands.md#dotenv-global-get) | :octicons-globe-16: global | The [global DDEV directory](../usage/architecture.md#global-files), DDEV v1.25.4+ |

```bash
ddev dotenv set .ddev/.env.redis --redis-tag 7-bookworm
ddev dotenv get .ddev/.env.redis --redis-tag
```

## See Also

* [Using Add-ons](../extend/using-add-ons.md#customizing-add-on-configuration) for configuring an add-on this way
* [Creating Add-ons](../extend/creating-add-ons.md) for shipping env files with an add-on
* [Custom Docker Compose Services](../extend/custom-docker-services.md) for using these variables in your own service
* [Environment Variables Provided](../extend/custom-commands.md#environment-variables-provided) for the `DDEV_*` variables DDEV sets itself
