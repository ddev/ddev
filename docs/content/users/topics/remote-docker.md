# Remote Docker Environments

## Remote Docker Instances

You can use remote Docker instances, whether on the internet, inside your network, or running in a virtual machine.

* On the remote machine, the Docker port must be exposed if it’s not already. See [instructions](https://gist.github.com/styblope/dc55e0ad2a9848f2cc3307d4819d819f) for how to do this on a systemd-based remote server. **Be aware that this has serious security implications and must not be done without taking those into consideration.** In fact, `dockerd` will complain:

    > Binding to IP address without `--tlsverify` is insecure and gives root access on this machine to everyone who has access to your network.  host="tcp://0.0.0.0:2375".

* If you do not already have the Docker client installed (like you would from Docker Desktop), install *only* the client with `brew install docker`.
* Create a Docker context that points to the remote Docker instance. For example, if the remote hostname is `debian-11`, then `docker context create debian-11 --docker host=tcp://debian-11:2375 && docker use debian-11`. Alternately, you can use the `DOCKER_HOST` environment variable, e.g. `export DOCKER_HOST=tcp://debian-11:2375`.
* Make sure you can access the remote machine using `docker ps`.
* Bind mounts cannot work on a remote Docker setup, so you must use `ddev config global --no-bind-mounts`. This will cause DDEV to push needed information to and from the remote Docker instance when needed. This also automatically turns on Mutagen caching.
* You may want to use a FQDN other than `*.ddev.site` because the DDEV site will *not* be at `127.0.0.1`. For example, `ddev config --fqdns=debian-11` and then use `https://debian-11` to access the site.
* If the Docker host is reachable on the internet, you can actually enable real HTTPS for it using Let’s Encrypt as described in [Casual Webhosting](../topics/hosting.md). Make sure port 2375 is not available on the internet.

## Continuous Integration (CI)

A number of people have found it easy to test their projects using DDEV on a CI system like [GitHub Actions](https://github.com/features/actions), [Travis CI](https://www.travis-ci.com), or [CircleCI](https://circleci.com). Instead of setting up a hosting environment for testing, they start the project using DDEV and run their tests.

Examples of this approach are demonstrated in [Codeception Tests in Travis CI with DDEV and Selenium](https://dev.to/tomasnorre/codeception-tests-in-travis-ci-with-ddev-and-selenium-1607) and [Setup DDEV in GitHub Workflows](https://github.com/marketplace/actions/setup-ddev-in-github-workflows).

## Integration of DDEV Docker Images Into Other Projects

You can use DDEV Docker images outside the context of the DDEV environment. People have used the `ddev-webserver` image for running tests in PhpStorm, for example.
