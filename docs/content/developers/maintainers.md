# Maintainer Tasks, Privileges, Resources

## THANK YOU

We so appreciate our amazing maintainers. There are so many things to keep track of, including support, testing, test runners, improvements, and a thousand other things. This section attempts to document some of the things that maintainers need to know and do.

## Maintainer Responsibilities

Not all maintainers can do all these things at any given time, but these are the things that we hope get done in the DDEV project:

* **Support**: We try to give friendly, accurate, and timely responses to those who need help in:
    * [Issue queue](https://github.com/ddev/ddev/issues) (and discussions, etc). Please follow all in at least the ddev/ddev project. On the Watch/Unwatch button at the top of the repository, consider selecting "All Activity". Also consider this on other projects in the `ddev` organization or other projects that are in your interest area.
    * Discord: Please read everything that happens in the [DDEV Discord](https://discord.gg/5wjP76mBJD) and respond to questions that you can help with.
    * Stack Overflow. You can subscribe to the [ddev tag on Stack Overflow](https://stackoverflow.com/questions/tagged/ddev) using the [email filter](https://meta.stackoverflow.com/a/400613/8097891) and answer or comment on questions there.
    * Often in [Drupal Slack](https://www.drupal.org/join-slack) #ddev channel. We have tried and tried to get people over to Discord, but it's still pretty active there.
    * Other add-on repositories or related repos where we can help.
* **Test Runner and Test System Maintenance**: The testing system is complex, and most tests are end-to-end tests, which can be fragile due to design, internet problems, changes upstream, etc. When something goes wrong, we want to figure out what it is and make it better. This can include debugging or rebooting Buildkite-runners, etc.
* **Test Maintenance**: Getting great tests that tell us what we need to know without taking forever and without being fragile is a never-ending battle. Improvements are always a goal. But it's fundamental that we be able to pass tests before we add new PRs in.
* **Improve the DDEV codebase with fixes and features**: We try to listen to the community and improve based on their needs.
* **Security Best Practices**: Please make sure to remain current on all security best practices. Your GitHub login and 1Password access absolutely must be managed with 2FA. Be aware of the fact that someone who compromises your privileges could attack our entire user base. If you have any questions about best practices, let's talk and make sure we all understand what's going on.

## Appropriate Use of Privileges

* We prefer the forked-PR workflow for all code changes. There are a few cases where a branch-PR on `ddev/ddev`, but in general, to do a fix or a feature, do it on a branch on your fork, and submit it as a forked PR.
* Even though you may have privileges to do things like push directly to the default branch of a repository, it doesn't mean you should use them. The vast majority of the time you'll use the codebase the same as any other contributor. PRs make it clear both now and in the future why changes were made.
* Use clear PRs and write great issues even though you yourself may understand exactly what's going on. Remember that you may need a refresher course in what you did in a month or a year, so write a great PR description and fill in the form.
* Remember to talk about configuration changes you make with other maintainers. Don't waste their time by changing things they'll then have to discover and debug.

## Maintainer Documentation Resources

* [DDEV Developer Documentation](./index.md)
* [Maintainer Private Repository](https://github.com/ddev/maintainer-info). This has information that may be sensitive, with screencasts and tips.
* [DDEV Contributor Training](https://ddev.com/blog/contributor-training/), a series of training sessions that were recorded.
* [DDEV blog](https://ddev.com/blog)
* 1Password passwords and tokens: Maintainers should be added to the DDEV team in 1Password. This gives access to the DDEV team vault, which has tokens and passwords that are needed for various things. Please try to maintain things like tokens in there.

## Privileges Required for Maintainers

Most privileges should be granted per-developer as separate accounts. In general, we don't want to share a common login. So for example, instead of sharing a login to [developer.apple.com](https://developer.apple.com) or [buildkite.com](https://buildkite.com/ddev) each maintainer should have their own login.

There are cases like access to hosting provider integrations that have essentially no value upstream where a shared login is acceptable. And of course, tokens listed in 1Password are a type of shared login. Our hosting integrations like Acquia, Platform.sh, etc. should never have any valuable things to attack anyway, so these should be very low risk. However, the bad guys are always trying new things...

* **GitHub**: Maintainers should usually be added to the [DDEV organization](https://github.com/orgs/ddev/people), usually was "owner", but lesser privileges are possible, and some maintainers may want only access to the DDEV project, etc.
* **Buildkite**: Maintainers should be added to the [DDEV Buildkite organization](https://buildkite.com/organizations/ddev/users) with "maintainer" privileges. This gives access to the Buildkite pipelines and the ability to add new pipelines. Do not require "SSO" or people won't be able to get in.
* **Chrome Remote Desktop**: This is the test runner login ("DDEV buildkite test-runners - remotedesktop.google.com") from 1Password, but it will need to be authorized via 2FA or a backup code from 1Password.
* **CircleCI**: Maintainers automatically have some access via their GitHub team membership, but should probably get more.
* **developer.apple.com**: Add to the DDEV team there so certificates can be managed.
* **hub.docker.com**: Add user to owners team in DDEV org.
* **Chocolatey**: Add user to [Manage maintainers](https://community.chocolatey.org/packages/ddev/ManagePackageOwners).
* **Read the Docs**: Add user to [Maintainers](https://readthedocs.org/dashboard/ddev/users/).
* **Icinga monitoring system**: This is documented in [maintainer-info](https://github.com/ddev/maintainer-info).
* **Discord**: Make admin in Discord.
* **Twitter (X)**: Posting is enabled by login in 1Password.
* **Mastodon**: Posting is enabled by login in 1Password.
* **Zoho Mail** is how `ddev.com` mail is routed; currently only Randy has an account, but we should consider adding others and making sure that more than one person can maintain it.
* **Zoho CRM** is how we track contacts and send monthly emails or announcements. People involved in marketing will want to have access to this, but it will cost for additional users.
* **[1Password](https://1password.com/)**. Maintainers should be added to the DDEV team in 1Password. This gives access to the DDEV team vault, which has tokens and passwords that are needed for various things. Please try to maintain things like tokens in there.
* Acquia Cloud test account
* Platform.sh test account
* Pantheon test account
* Lagoon test account
* [Newmonitor.thefays.us](https://newmonitor.thefays.us/icingaweb2/dashboard) (Test runner monitoring).
* SSH (and sudo) access to `newmonitor.thefays.us`
* **[developer.apple.com](https://developer.apple.com)** - Maintainers should be added to the DDEV team in the Apple Developer program, so that they can create new certificates.
* **SSH access to newmonitor.thefays.us**.
* **Account on pi.ddev.site**.
* **Notifications from newmonitor.thefays.us**.
* **Web access to newmonitor.thefays.us**.
* **SSH access to behind-firewall monitoring proxy**.
* **Amplitude**: Invite new user at [team management](https://analytics.amplitude.com/ddev/settings/team).
* **Stack Overflow**: Follow the [ddev tag on Stack Overflow](https://stackoverflow.com/questions/tagged/ddev) and answer or comment on questions there when possible.

## Newmonitor.thefays.us use and maintenance

[Newmonitor.thefays.us](https://newmonitor.thefays.us) is an Icinga instance that monitors our Buildkite test runners and a few other things like [ddev.com](https://ddev.com), etc. It also monitors some of Randy's small sites, but those can be ignored.

Maintainers have a login to [the dashboard](https://newmonitor.thefays.us/icingaweb2/dashboard) and should receive emails when problems are discovered.

You can quickly check [the dashboard](https://newmonitor.thefays.us/icingaweb2/dashboard) to see the current status if you get an email notification. Sometimes the tests are flaky, and of course during power outages or internet outages there may be un-resolvable items.

## Test Runner Maintenance

* When you change things on a test runner, or you solve a problem, or reboot it, add a comment to [ddev/maintainer-info/issues/1](https://github.com/ddev/maintainer-info/issues/1) so others will know what's going on.
