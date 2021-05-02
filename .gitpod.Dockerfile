FROM gitpod/workspace-full
SHELL ["/bin/bash", "-c"]

RUN sudo apt-get update && sudo apt-get install -y file mysql-client netcat telnet
RUN brew update && brew install bash-completion bats-core drud/ddev/ddev golangci-lint
RUN npm install -g markdownlint-cli

RUN echo 'if [ -r "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh" ]; then . "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh"; fi' >>~/.bashrc

RUN echo 'export PATH=~/bin:$PATH' >>~/.bashrc && mkdir -p ~/bin
RUN ln -sf /workspace/ddev/.gotmp/bin/linux_amd64/ddev ~/bin/ddev

RUN curl -o ~/bin/gitpod-setup-ddev.sh --fail -lLs https://raw.githubusercontent.com/shaal/ddev-gitpod/main/.ddev/gitpod-setup-ddev.sh && chmod +x ~/bin/gitpod-setup-ddev.sh
RUN mkcert -install

ENV BUILDX_BINARY_URL="https://github.com/docker/buildx/releases/download/v0.5.1/buildx-v0.5.1.linux-amd64"

RUN mkdir -p ~/.docker/cli-plugins && curl --output ~/.docker/cli-plugins/docker-buildx \
    --silent --show-error --location --fail --retry 3 \
    "$BUILDX_BINARY_URL" && chmod a+x ~/.docker/cli-plugins/docker-buildx

# Install custom tools, runtimes, etc.
# For example "bastet", a command-line tetris clone:
# RUN brew install bastet
#
# More information: https://www.gitpod.io/docs/config-docker/

###
### Initiate a rebuild of Gitpod's image by updating this comment #1
###
