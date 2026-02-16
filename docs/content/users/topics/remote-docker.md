# Remote Docker Environments

## Remote Docker Instances

You can use remote Docker instances, whether on the internet, inside your network, or running in a virtual machine. This lets you offload Docker resource usage from your local machine, which can improve battery life and performance on less powerful devices.

### Connecting to the Remote Docker Host

There are two main ways to connect to a remote Docker instance:

#### SSH-Based Docker Context (Recommended)

The SSH approach is more secure because it uses your existing SSH authentication and does not require exposing the Docker port on the network:

```bash
docker context create remotedocker --docker "host=ssh://user@remote-host"
docker context use remotedocker
```

#### TCP-Based Docker Context

You can expose the Docker API over TCP, but **be aware that this has serious security implications**. See [instructions](https://gist.github.com/styblope/dc55e0ad2a9848f2cc3307d4819d819f) for how to do this on a systemd-based remote server. `dockerd` will warn:

> Binding to IP address without `--tlsverify` is insecure and gives root access on this machine to everyone who has access to your network.  host="tcp://0.0.0.0:2375".

```bash
docker context create remote --docker "host=tcp://remote-host:2375"
docker context use remote
```

If you do not already have the Docker client installed (like you would from Docker Desktop), install *only* the client with `brew install docker`.

Verify your connection with `docker ps`.

### Required DDEV Configuration

* **Disable bind mounts**: Bind mounts cannot work with a remote Docker setup, so you must use `ddev config global --no-bind-mounts`. This causes DDEV to push needed information to and from the remote Docker instance when needed. This also automatically turns on Mutagen caching.

### Accessing Your Sites

The DDEV site will *not* be at `127.0.0.1`. You have two options:

* **SSH tunnel**: Forward DDEV ports (like 8080, 8443) from the remote host to your local machine. This lets you access sites at `localhost:<port>` without exposing ports on the remote host.
* **Direct access with custom FQDN**: Use `ddev config --additional-fqdns=remote-host` and access the site at `https://remote-host`. If the Docker host is reachable on the internet, you can enable real HTTPS using Let's Encrypt as described in [Hosting with DDEV](../topics/hosting.md). Make sure port 2375 is not available on the internet.

## Continuous Integration (CI)

A number of people have found it easy to test their projects using DDEV on a CI system like GitHub Actions, Travis CI, or CircleCI. Instead of setting up a hosting environment for testing, they start the project using DDEV and run their tests.

Examples of this approach are demonstrated in [Codeception Tests in Travis CI with DDEV and Selenium](https://dev.to/tomasnorre/codeception-tests-in-travis-ci-with-ddev-and-selenium-1607) and [Setup DDEV in GitHub Workflows](https://github.com/marketplace/actions/setup-ddev-in-github-workflows).

## Integration of DDEV Docker Images Into Other Projects

You can use DDEV Docker images outside the context of the DDEV environment. People have used the `ddev-webserver` image for running tests in PhpStorm, for example.
