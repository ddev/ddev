# Writing Style Guide

This page formalizes the writing conventions we aspire to use in the documentation.  
It’s a living statement of intent and a reference for all contributors.

## Voice and Tone

In the spirit of the [Code of Conduct](../users/code-of-conduct.md), we want to be clear and encouraging for everyone that bothers to read DDEV’s documentation, rewarding the time and attention they choose to give to it.

### Beginner-Friendly, Expert-Compatible

Write so a DDEV beginner can follow your guidance and a DDEV veteran could use the same content as a reference.

<!-- textlint-disable -->

| Write This 👍 | Not This ❌
| -- | --
| You can run `ddev restart` to apply changes you’ve made to your global `~/.ddev/global_config.yaml` or per-project `.ddev/config.yaml`. | Just `ddev restart` to apply YAML config changes.

<!-- textlint-enable -->

### Be Efficient and Direct

Brevity makes for good documentation!

Few read the docs from start to finish like a book, and explanation can be a barrier to learning. Get to the point, avoiding parentheticals and mid-sentence notes that interrupt the main flow.

Omit extraneous explanation or decorative language that doesn’t help the reader. Instructions don’t need to include “please”. Provide some context for anchor links without overloading them to hinder flow.

| Write This 👍 | Not This ❌
| -- | --
| Run `ddev start` and launch the site in a browser. | Please run `ddev start`, then launch the site in a browser.
| Learn more on the [Extending](../users/extend/customization-extendibility.md) page. | (You can also learn more about this and related topics in [Providing Custom Environment Variables to a Container](../users/extend/customization-extendibility.md).)

### Avoid “Just” and “Easy”

<!-- textlint-disable -->

Try not to use language that [may talk down to the reader](https://justsimply.dev/). You may intend for “it’s easy” to be reassuring, but it’s a subjective judgment that can convince someone struggling that they’re doing it wrong. Things could instead be “straightforward” if they’re without nuance, “simple” if they don’t involve complex actions or concepts, or “quick” if they involve one or two steps that’d be fast even on someone’s worst day with the slowest-imaginable machine.

Similarly, “just do X” suggests that “X” should be easy or obvious. Most of the time “just” can be omitted and everyone wins.

<!-- textlint-enable -->

If you’d like to reassure the reader something is easy, illustrate it with a demonstration and let them draw their own conclusion!

<!-- textlint-disable -->

| Write This 👍 | Not This ❌
| -- | --
| Change your project’s PHP version by either editing `.ddev/config.yaml` to set `php_version: "8.2"`, or by running `ddev config --php-version=8.2`, followed by running `ddev restart`. | It’s easy to change your project’s PHP version! Just edit your project’s `.ddev/config.yaml` to set `php_version: "8.2"`, or run `ddev config --php-version=8.2`, followed by running `ddev restart`. |

<!-- textlint-enable -->

## Writing Style

DDEV’s documentation should be consistent throughout, which benefits both the reader taking in information and the contributor looking for examples to follow.

!!!tip "Read It Aloud"
    If you get tripped up speaking your words out loud, someone else will get tripped up reading them, too.

### Use Correct Capitalization and Punctuation

Write with appropriate grammar and style for U.S. English, including capitalization and punctuation. Variations in spelling and writing style make the documentation harder to read, and we want to be respectful of the reader’s time and attention.

<!-- textlint-disable -->

| Write This 👍 | Not This ❌
| -- | --
| Let’s not capitalize random words for emphasis. | Let’s not Capitalize random Words for emphasis.
| That organization uses a lovely American color. | That organisation uses a lovely American colour.
| We can use “curly quotes” now that we’re post-typewriter. | We can use "curly quotes" now that we're post-typewriter.

<!-- textlint-enable -->

### “Run” Commands

We “run” commands. We don’t “do” them, and the command itself is not a verb. Whenever possible, reinforce that a given thing in backticks is intended as a console command by using the word “run” before it.

<!-- textlint-disable -->

| Write This 👍 | Not This ❌ |
| -- | -- |
| Run `ddev config` to set up your project. | Do `ddev config` to set up your project.<br>You can `ddev config` to set up your project. |
| If you get stuck, run `ddev restart`. | If you get stuck, just `ddev restart`. |

<!-- textlint-enable -->

### Use Active Third Person

Avoid impersonal language featuring unknown individuals or shadowy organizations.  
“It is recommended,” for example, could be a warmer “we recommend” or “Laravel users recommend”.

Write on behalf of the community and not yourself—use “we” and not “I”.

<!-- textlint-disable -->

| Write This 👍 | Not This ❌ |
| -- | -- |
| We recommend Colima for the best performance. | It is recommended for performance that you use Colima. |
| Some prefer Redis for runtime caches. | I like using Redis for runtime caches. |

<!-- textlint-enable -->

### Write Once and Link

Try to keep from repeating yourself in the documentation. Instead, write carefully and link to that well-crafted specimen, whether it’s across the page or off to another section. This has two benefits:

1. Easier maintenance with less chance of redundant information becoming stale.
2. Subtle reinforcement of documentation structure that helps the reader learn where to find answers, rather than answering the same thing in different places.

### Mind Your Context

It’s easy to get lost in documentation; don’t assume the reader is always following your words. Take care to bring the reader with you, especially if there are steps that involve different applications or distinct areas of concern.

<!-- textlint-disable -->

| Write This 👍 | Not This ❌ |
| -- | -- |
| 1. In Docker Desktop, go to *Resources* → *Advanced* and set “Memory” to 6GB.<br>2. From your terminal, run `ddev restart`.<br>3. In your text editor, open `.ddev/config.yaml` and set `php_version: "8.1"`. | 1. Go to *Resources* → *Advanced* and set “Memory” to 6GB.<br>2. Run `ddev restart`.<br>3. Set `php_version: "8.1"`. |
| Once you’ve [installed a Docker provider](../users/install/docker-installation.md), you’re ready to install DDEV! | Docker or an alternative is required before anything will work with DDEV. This is pretty easy on most environments; see the [Docker Installation](../users/install/docker-installation.md) page to help sort out the details.

<!-- textlint-enable -->

### Avoid Starting with Asides

The beginning of a page or section should introduce what the rest of the content is about. Try to avoid starting with asides or reminders that get in the way of this initial statement of purpose.

Never tell the reader to “remember” something they may not have been introduced to yet.

### Use Tips

Avoid using “Note:” to signal an aside. Most sentences work fine without it, and for discreet notes we have `!!!note`, `!!!tip`, and `!!!warning` conventions.

Use one of these callouts for text that can stand on its own and be skipped, or for an urgent message that needs greater visual emphasis.

Summarize the callout’s contents with a succinct heading whenever you can, so anyone skimming can know whether to read the callout’s supporting text.

```
!!!note "This is a note."
    Use it for extraneous asides.

!!!tip "This is a tip."
    Use it for helpful asides.

!!!warning "This is a warning."
    Use it for asides that should have urgent emphasis.

!!!note
    This is a note without a heading, which should only be used with the author isn’t clever enough to come up with a succinct one. (The “Note” is added automatically.)
```

!!!note "This is a note."
    Use it for extraneous asides.

!!!tip "This is a tip."
    Use it for helpful asides.

!!!warning "This is a warning."
    Use it for asides that should have urgent emphasis.

!!!note
    This is a note without a heading, which should only be used with the author isn’t clever enough to come up with a succinct one. (The “Note” is automatically added.)

**Note:** we want to avoid callouts like this sentence, that should either be tips or flow naturally with their surrounding text. If any documentation *shouldn’t* be noted by the reader, get rid of it.

### Use Correct Proper Nouns

#### DDEV != `ddev`

DDEV is a product and `ddev` is a binary or console command. DDEV should always be uppercase, and `ddev` should always be in backticks. DDEV-Local and DDEV-Live are former product incarnations that shouldn’t be found in modern documentation.

<!-- textlint-disable -->

| Write This 👍 | Not This ❌
| -- | --
| DDEV is wonderful! | Ddev is wonderful!<br>ddev is wonderful!<br>DDEV-Local is wonderful!
| Run `ddev`. | Run ddev.<br>Run DDEV.

<!-- textlint-enable -->

#### Products, Organizations, and Protocols

When in doubt, honor whatever name a product or organization uses in its official materials.  
Use backticks to differentiate between a product and command, like DDEV vs. `ddev`.

<!-- textlint-disable -->

| Write This 👍 | Not This ❌
| -- | --
| AMD64, ARM64, and Apple Silicon | amd64, arm64, M1 Macs
| Apache | apache
| Bash or `bash` | bash
| Blackfire | blackfire, Blackfire.io
| Chocolatey | chocolatey
| Colima or `colima` | colima
| Composer or `composer` | composer
| Docker or `docker` | docker
| Drush or `drush` | drush
| Git | git
| Git Bash | git bash
| GitHub or `github` | Github
| Gitpod or `gitpod` | Gitpod.io, GitPod
| GoLand | Goland
| Google | google
| Homebrew | homebrew
| HTTP, HTTPS, SSH, `http`, `https`, `ssh` | http, https, ssh
| IPv4, IPv6 | IPV4, IPV6, ipv4, ipv6
| Linux | linux
| nginx or `nginx` | Nginx, NGINX
| Node.js or `node` | Node, node
| NFS | nfs
| Pantheon | pantheon, Pantheon.io
| PHP or `php` | php
| PhpStorm | PHPStorm, PHPstorm, Phpstorm
| PHPUnit or `phpunit` | phpunit, PHPunit
| PostgreSQL | Postgres
| Terminus | terminus
| Windows | windows
| Xdebug | XDebug, xDebug

<!-- textlint-enable -->

### Quote Copied Text

If you’re quoting a human being or a message lifted verbatim from some other source (outside a fenced code block), make sure it ends up in a `<blockquote>` element:

```
> Error: your quote style should not always be in a fenced block.
```

> Error: your quote style should not always be in a fenced block.

### Other Recommendations

One-off tips that don’t fit nicely into any of the sections above:

<!-- textlint-disable -->

- Pluralize “CMS” as “CMSes”, not “CMSs”.
- Use all-caps references for file *types* like JSON, YAML and CSS.
- Wrap file *extensions* in backticks like `.json`, `.yaml`, and `.css`.
- Wrap references to files, directories, images and commands in backticks.
- Use Title Case for headings wherever it makes sense.
- Link to related services and topics where convenient—usually first use on a given page.
- Use `<kbd>` elements for representing literal keystrokes.
- Use sequential numbers for numbered lists in the source Markdown, regardless of how they’re eventually rendered.
- Try to maintain parallel format for list items.

| Write This 👍 | Not This ❌ |
| -- | -- |
| web server | webserver
| add-on | addon
| JSON, YAML, CSS | json, Yaml, css
| `.json`, `.yaml`, `.css`, `~/.ddev` | .json, .yaml, .css, ~/.ddev
| <kbd>CTRL</kbd> + <kbd>C</kbd> | control-c, control + c, ctrl+c
| *Menu Item* → *Another Menu Item* → *Setting* | Menu Item>Another Menu Item>Setting<br>Menu Item -> Another Menu Item -> Setting
| several CMSes | several CMSs, several CMS’s
| How to Reticulate Splines | How to reticulate splines
| 1. Run `command`.<br>2. Edit file.<br>3. Restart computer. | 1. `command`<br>2. Edit file.<br>3. Additionally, restart your computer.

<!-- textlint-enable -->
