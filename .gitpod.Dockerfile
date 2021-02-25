FROM gitpod/workspace-full
SHELL ["/bin/bash", "-c"]
# Install ddev
RUN brew update && brew install bash-completion drud/ddev/ddev golangci-lint

RUN echo 'if [ -r "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh" ]; then . "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh"' >>~/.bash_profile

RUN brew unlink ddev && ln -sf ~/workspace/ddev/.gotmp/bin/linux_amd64/ddev /usr/local/bin/ddev && chmod 777 /usr/local/bin/ddev

RUN mkcert -install

# Install custom tools, runtimes, etc.
# For example "bastet", a command-line tetris clone:
# RUN brew install bastet
#
# More information: https://www.gitpod.io/docs/config-docker/
