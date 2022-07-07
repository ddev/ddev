# Copyright (c) 2016-2022 Martin Donath <martin.donath@squidfunk.com>

# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to
# deal in the Software without restriction, including without limitation the
# rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
# sell copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:

# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NON-INFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
# FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
# IN THE SOFTWARE.

import logging
import os
import posixpath
import re
import requests

from fnmatch import fnmatch
from lxml import html
from mkdocs import utils
from mkdocs.commands.build import DuplicateFilter
from mkdocs.config.config_options import Choice, Deprecated, Type
from mkdocs.plugins import BasePlugin
from pathlib import Path
from urllib.parse import urlparse

# -----------------------------------------------------------------------------
# Class
# -----------------------------------------------------------------------------

# Privacy plugin
class PrivacyPlugin(BasePlugin):

    # Configuration scheme
    config_scheme = (
        ("enabled", Type(bool, default = True)),
        ("cache_dir", Type(str, default = ".cache/plugin/privacy")),

        # Options for external assets
        ("externals", Choice(("bundle", "report"), default = "bundle")),
        ("externals_dir", Type(str, default = "assets/externals")),
        ("externals_exclude", Type(list, default = [])),

        # Deprecated options
        ("download", Deprecated(moved_to = "enabled")),
        ("download_directory", Deprecated(moved_to = "externals_dir")),
        ("externals_directory", Deprecated(moved_to = "externals_dir")),
    )

    # Determine base URL and directory
    def on_config(self, config):
        self.base_url = urlparse(config.get("site_url"))
        self.base_dir = config["site_dir"]
        self.cache = self.config["cache_dir"]
        self.files = []

    # Determine files that need to be post-processed
    def on_files(self, files, config):
        if not self.config["enabled"]:
            return

        # Filter relevant files, short-circuit Lunr.js
        for file in files:
            if file.url.endswith(".js") or file.url.endswith(".css"):
                if not "assets/javascripts/lunr" in file.url:
                    self.files.append(file)

        # If site URL is not given, add Mermaid.js - see https://bit.ly/36tZXsA
        # This is a special case, as Material for MkDocs automatically loads
        # Mermaid.js when a Mermaid diagram is found in the page.
        if not config.get("site_url"):
            if not any("mermaid" in js for js in config["extra_javascript"]):
                config["extra_javascript"].append(
                    "https://unpkg.com/mermaid@9.0.1/dist/mermaid.min.js"
                )

    # Parse, fetch and store external assets in pages
    def on_post_page(self, output, page, config):
        if not self.config["enabled"]:
            return

        # Find all external resources
        expr = re.compile(
            r'<(?:link[^>]+href?|(?:script|img)[^>]+src)=[\'"]?http[^>]+>',
            re.IGNORECASE | re.MULTILINE
        )

        # Parse occurrences and replace in reverse
        for match in reversed(list(expr.finditer(output))):
            value = match.group()

            # Compute offsets for replacement
            l = match.start()
            r = l + len(value)

            # Handle preconnect hints and style sheets
            el = html.fragment_fromstring(value)
            if el.tag == "link":
                raw = el.get("href", "")

                # Check if URL is external
                url = urlparse(raw)
                if not self._is_external(url):
                    continue

                # Replace external preconnect hint in output
                rel = el.get("rel")
                if rel == "preconnect":
                    output = output[0:l] + output[r:]

                # Replace external style sheet in output
                if rel == "stylesheet":
                    output = "".join([
                        output[:l],
                        value.replace(raw, self._fetch(url, page)),
                        output[r:]
                    ])

            # Handle external scripts and images
            if el.tag == "script" or el.tag == "img":
                raw = el.get("src", "")

                # Check if URL is external
                url = urlparse(raw)
                if not self._is_external(url):
                    continue

                # Replace external script or image in output
                output = "".join([
                    output[:l],
                    value.replace(raw, self._fetch(url, page)),
                    output[r:]
                ])

        # Return output with replaced occurrences
        return output

    # Parse, fetch and store external assets in assets
    def on_post_build(self, config):
        if not self.config["enabled"]:
            return

        # Check all files that are part of the build
        for file in self.files:
            full = os.path.join(self.base_dir, file.dest_path)
            if not os.path.isfile(full):
                continue

            # Handle internal style sheet or script
            if full.endswith(".css") or full.endswith(".js"):
                with open(full, encoding = "utf-8") as f:
                    utils.write_file(
                        self._fetch_dependents(f.read(), file.dest_path),
                        full
                    )

    # -------------------------------------------------------------------------

    # Check if the given URL is external
    def _is_external(self, url):
        return url.hostname != self.base_url.hostname

    # Check if the given URL is excluded
    def _is_excluded(self, url, base):
        url = re.sub(r'^https?:\/\/', "", url)
        for pattern in self.config["externals_exclude"]:
            if fnmatch(url, pattern):
                log.debug(f"Excluding external file in '{base}': {url}")
                return True

        # Exclude all external assets if bundling is not enabled
        if self.config["externals"] == "report":
            log.warning(f"External file in '{base}': {url}")
            return True

    # Fetch external resource in given page
    def _fetch(self, url, page):
        raw = url.geturl()

        # Check if URL is excluded
        if self._is_excluded(raw, page.file.dest_path):
            return raw

        # Download file if it's not contained in the cache
        path = file = os.path.join(self.cache, self._resolve(url))
        if not os.path.isfile(file):
            log.debug(f"Downloading external file: {raw}")
            res = requests.get(raw, headers = {

                # Set user agent explicitly, so Google Fonts gives us *.woff2
                # files, which according to caniuse.com is the only format we
                # need to download as it covers the entire range of browsers
                # we're officially supporting
                "User-Agent": " ".join([
                    "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
                    "AppleWebKit/537.36 (KHTML, like Gecko)",
                    "Chrome/98.0.4758.102 Safari/537.36"
                ])
            })

            # Compute and ensure presence of file extension
            name = re.findall(r'^[^;]+', res.headers["content-type"])[0]
            extension = extensions.get(name)
            if extension and not file.endswith(extension):
                file = str(Path(file).with_suffix(extension))

            # Write contents and create symbolic link if necessary
            utils.write_file(res.content, file)
            if path != file:

                # Creating symbolic links might fail on Windows. Thus, we just
                # print a warning and continue - see https://bit.ly/3xYFzcZ
                try:
                    os.symlink(os.path.basename(file), path)
                except OSError:
                    log.warning(f"Couldn't create symbolic link '{file}'")

        # Append file extension from file after resolving symbolic links
        _, extension = os.path.splitext(os.path.realpath(file))
        if not file.endswith(extension):
            file = str(Path(file).with_suffix(extension))

        # Compute final path relative to output directory
        path = file.replace(self.cache, self.config["externals_dir"])
        full = os.path.join(self.base_dir, path)
        if not os.path.exists(full):

            # Open file and patch dependents resources
            if extension == ".css" or extension == ".js":
                with open(file, encoding = "utf-8") as f:
                    utils.write_file(
                        self._fetch_dependents(f.read(), path),
                        full
                    )

            # Copy file from cache to output directory
            else:
                utils.copy_file(file, full)

        # Return URL relative to current page
        return utils.get_relative_url(
            utils.normalize_url(path),
            page.url
        )

    # Fetch dependent resources in external assets
    def _fetch_dependents(self, output, base):

        # Fetch external assets in style sheet
        if base.endswith(".css"):
            expr = re.compile(
                r'url\((\s*http?[^)]+)\)',
                re.IGNORECASE | re.MULTILINE
            )

        # Fetch external assets in script
        elif base.endswith(".js"):
            expr = re.compile(
                r'["\'](http[^"\']+\.js)["\']',
                re.IGNORECASE | re.MULTILINE
            )

        # Parse occurrences and replace in reverse
        for match in reversed(list(expr.finditer(output))):
            value = match.group(0)
            raw   = match.group(1)

            # Compute offsets for replacement
            l = match.start()
            r = l + len(value)

            # Check if URL is external
            url = urlparse(raw)
            if not self._is_external(url):
                continue

            # Check if URL is excluded
            if self._is_excluded(raw, base):
                continue

            # Download file if it's not contained in the cache
            file = os.path.join(self.cache, self._resolve(url))
            if not os.path.isfile(file):
                log.debug(f"Downloading external file: {raw}")
                res = requests.get(raw)
                utils.write_file(res.content, file)

            # Compute final path relative to output directory
            path = os.path.join(
                self.config["externals_dir"],
                self._resolve(url)
            )

            # Create relative URL for asset in style sheet
            if base.endswith(".css"):
                url = utils.get_relative_url(path, base)

            # Create absolute URL for asset in script
            elif base.endswith(".js"):
                url = posixpath.join(self.base_url.geturl(), path)

            # Replace external asset in output
            output = "".join([
                output[:l],
                value.replace(raw, url),
                output[r:]
            ])

            # Copy file from cache to output directory
            full = os.path.join(self.base_dir, path)
            utils.copy_file(file, full)

        # Return output with replaced occurrences
        return bytes(output, encoding = "utf8")

    # Resolve filename with respect to operating system
    def _resolve(self, url):
        data = url._replace(scheme = "", query = "", fragment = "")
        return os.path.sep.join(data.geturl()[2:].split("/"))

# -----------------------------------------------------------------------------
# Data
# -----------------------------------------------------------------------------

# Set up logging
log = logging.getLogger("mkdocs")
log.addFilter(DuplicateFilter())

# Expected file extensions
extensions = dict({
    "application/javascript": ".js",
    "image/avif": ".avif",
    "image/gif": ".gif",
    "image/jpeg": ".jpg",
    "image/png": ".png",
    "image/svg+xml": ".svg",
    "image/webp": ".webp",
    "text/javascript": ".js",
    "text/css": ".css"
})
