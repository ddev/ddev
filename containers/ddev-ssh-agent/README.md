# ddev-ssh-agent

originally forked from <https://github.com/nardeas/docker-ssh-agent>
at `fb6822d0003d1c0a795e183f5d257c2540fa74a4`.

# Docker SSH Agent

[![Pulls](https://img.shields.io/docker/pulls/nardeas/ssh-agent.svg)](https://img.shields.io/docker/pulls/nardeas/ssh-agent.svg?style=flat-square)
[![Size](https://images.microbadger.com/badges/image/nardeas/ssh-agent.svg)](https://microbadger.com/images/nardeas/ssh-agent "Get your own image badge on microbadger.com")

Lets you store your SSH authentication keys in a dockerized ssh-agent that can provide the SSH authentication socket for other containers. Works in macOS and Linux environments.

## Why?

On macOS you cannot simply forward your authentication socket to a Docker container to be able to e.g clone private repositories that you have access to. You don't want to copy your private key to all containers either. The solution is to add your keys only once to a long-lived ssh-agent container that can be used by other containers and stopped when not needed anymore.

## hub.docker.com

You can pull the image from [DockerHub](https://hub.docker.com/r/nardeas/ssh-agent/) via

```
docker pull nardeas/ssh-agent
```

## How to use

### Quickstart

If you don't want to build your own images, here's a 3-step guide:

1\. Run agent

```
docker run -d --name=ssh-agent nardeas/ssh-agent
```

2\. Add your keys

```
docker run --rm --volumes-from=ssh-agent -v ~/.ssh:/.ssh -it nardeas/ssh-agent ssh-add /root/.ssh/id_rsa
```

3\. Now run your actual container:

```
docker run -it --volumes-from=ssh-agent -e SSH_AUTH_SOCK=/.ssh-agent/socket ubuntu:latest /bin/bash
```

**Run script**

You can run the `run.sh` script which will build the images for you, launch the ssh-agent and add your keys. If your keys are password protected (hopefully) you will just need to input your passphrase.

Launch everything:

```
./run.sh
```

Remove your keys from ssh-agent and stop container:

```
./run.sh -s
```

### Step by step

#### 0. Build

Navigate to the project directory and launch the following command to build the image:

```
docker build -t docker-ssh-agent:latest -f Dockerfile .
```

#### 1. Run a long-lived container

```
docker run -d --name=ssh-agent docker-ssh-agent:latest
```

#### 2. Add your ssh keys

Run a temporary container with volume mounted from host that includes your SSH keys. SSH key id_rsa will be added to ssh-agent (you can replace id_rsa with your key name):

```
docker run --rm --volumes-from=ssh-agent -v ~/.ssh:/.ssh -it docker-ssh-agent:latest ssh-add /root/.ssh/id_rsa
```

The ssh-agent container is now ready to use.

#### 3. Add ssh-agent socket to other container

If you're using `docker-compose` this is how you forward the socket to a container:

```
  volumes_from:
    - ssh-agent
  environment:
    - SSH_AUTH_SOCK=/.ssh-agent/socket
```

##### For non-root users

The above only works for root. ssh-agent socket is accessible only to the user which started this agent or for root user. So other users don't have access to `/.ssh-agent/socket`. If you have another user in your container you should do the following:

1. Install `socat` utility in your container
2. Make proxy-socket in your container:

```
sudo socat UNIX-LISTEN:~/.ssh/socket,fork UNIX-CONNECT:/.ssh-agent/socket &
```

3. Change the owner of this proxy-socket

```
sudo chown $(id -u) ~/.ssh/socket
```

4. You will need to use different SSH_AUTH_SOCK for this user:

```
SSH_AUTH_SOCK=~/.ssh/socket
```

##### Without docker-compose

Here's an example how to run a Ubuntu container that uses the ssh authentication socket:

```
docker run -it --volumes-from=ssh-agent -e SSH_AUTH_SOCK=/.ssh-agent/socket ubuntu:latest /bin/bash
```

### Deleting keys from the container

Run a temporary container and delete all known keys from ssh-agent:

```
docker run --rm --volumes-from=ssh-agent -it docker-ssh-agent:latest ssh-add -D
```
