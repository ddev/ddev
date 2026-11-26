---
name: ddev
description: Use DDEV, the Docker-based local development environment, in a project that has a .ddev directory. Use whenever you run PHP, Composer, Node.js, npm, Drush, WP-CLI, Artisan, a database client, or tests in such a project, and when DDEV itself misbehaves.
---
<!-- #ddev-generated: written by "ddev utility agent-skill install" from DDEV @@DDEV_VERSION@@. Running that command again replaces this file. To keep your own edits, delete this line, and DDEV leaves the file alone. -->

# Use DDEV

This project runs in DDEV: PHP, the web server, the database, Node.js, and
the project's tools run in Docker containers, not on the host. A `.ddev/`
directory at the project root holds its configuration.

Don't rely on this file for flags or options, because they change between
DDEV versions. Ask the CLI instead:

- `ddev help` lists the commands, including the project's custom commands.
- `ddev <command> --help` shows one command's flags and examples.
- `ddev --version` shows the DDEV version in use.

## Run tools through DDEV, not on the host

The host may have a different PHP, Composer, or Node.js version, or none.
Run the project's tools through `ddev` so they use the project's versions,
extensions, database, and environment:

- `ddev composer ...`, `ddev npm ...`, `ddev npx ...`, `ddev yarn ...`
- `ddev php ...`, `ddev drush ...`, `ddev wp ...`, `ddev artisan ...`
  (`php artisan`), `ddev console ...` (Symfony), `ddev craft ...`
- `ddev mysql`, `ddev mariadb` or `ddev psql` for the database
- anything else: `ddev exec <command>` runs it in the web container, where
  the project is mounted at `/var/www/html`; `ddev exec -s db <command>`
  runs it in the database container.

Host paths don't exist in the container: `/home/me/project/web/index.php`
on the host is `/var/www/html/web/index.php` there. Give `ddev exec` and
the tools paths relative to the project root, which is the default working
directory: `ddev php web/index.php`.

Run `ddev help` to see which of these this project provides. Don't install
PHP, Composer, Node.js, or a database on the host to work around a failure;
find out why the DDEV command failed.

## Quoting with `ddev exec`

`ddev exec` runs its arguments through Bash in the container, so the host
shell and the container shell each parse the command. Simple commands work
as written: `ddev exec ls -la web`. For pipes, redirects, variables, or
nested quotes, either:

- pass the whole command as one single-quoted argument, which the host
  shell leaves alone and the container's Bash runs as written:
  `ddev exec 'grep -r "TODO" src | wc -l'`, or
- write a script in the project and run it:
  `ddev exec bash scripts/task.sh`. This is the safer choice for anything
  long.

`ddev exec --raw -- <command> <args>` skips the container's Bash and passes
each argument unchanged. Run `ddev exec --help` for `--dir`, `--service`,
and `--user`.

## Commands with no equivalent in the container

These manage containers or touch the host, so nothing inside the container
can do their job: `ddev start`, `ddev stop`, `ddev restart`, `ddev poweroff`,
`ddev config`, `ddev add-on`, `ddev import-db`, `ddev export-db`,
`ddev import-files`, `ddev snapshot`, `ddev pull`, `ddev push`,
`ddev launch`, `ddev share`, `ddev delete`, and `ddev utility rebuild`.
After changing `.ddev/config.yaml` or a file in `.ddev/`, run
`ddev restart` for it to take effect.

Don't edit files that contain `#ddev-generated`, such as
`settings.ddev.php`, `wp-config-ddev.php`, or generated files in `.ddev/`,
because DDEV rewrites them. Put changes in `.ddev/config.yaml`, a new
`.ddev/config.*.yaml`, or a new file, see
<@@DOCS_URL@@/users/extend/customization-extendibility/>.

## When you are inside the web container

If `IS_DDEV_PROJECT=true` is set and the `ddev` command isn't found, you
are running inside the DDEV web container, for example in a terminal
opened with `ddev ssh`. There, run the tools directly (`composer`, `npm`,
`drush`, `php`) instead of through `ddev`. Commands that need the host,
such as `ddev restart` or `ddev import-db`, don't work there: stop and ask
the user to run them on the host.

## Machine-readable output

Add `-j` (`--json-output`) for JSON instead of parsing tables:

- `ddev describe -j` shows the project's URLs, services, ports, PHP and
  database versions, and status.
- `ddev list -j` lists all projects and their status.

Use the project's URLs from `ddev describe` rather than guessing them.

## Documentation

The DDEV docs are published as Markdown for agents:

- <@@DOCS_URL@@/llms.txt> lists each page with a link to its Markdown
  version. Read it, then fetch only the page you need.
- <@@DOCS_URL@@/llms-full.txt> has every page in one file. It is about
  1 MB, too large to read whole; search it instead.

These are published for the `stable` (latest release) and `latest`
(development) docs only, not for each release. If `ddev --version` is
older than the current release, a documented feature may not exist yet;
check with `ddev <command> --help`.

## When DDEV itself fails

- `ddev utility diagnose` checks Docker, networking, HTTPS, and the current
  project, and suggests fixes.
- `ddev logs` shows the web container's logs; `ddev logs -s db` the
  database's.
- `ddev utility` lists the other diagnostic commands (`port-diagnose`,
  `tls-diagnose`, `xdebug-diagnose`, `mutagen-diagnose`, and more).
- `ddev utility test` writes a full report to attach to an issue.
- Compare `ddev --version` with the latest release at
  <https://github.com/ddev/ddev/releases>. If the problem is fixed in a
  newer release, tell the user and link the release notes; upgrading DDEV
  is the user's decision.
- Search existing issues at <https://github.com/ddev/ddev/issues> before
  suggesting a new one.
