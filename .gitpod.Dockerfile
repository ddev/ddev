FROM gitpod/workspace-full
SHELL ["/bin/bash", "-c"]
# Install ddev
RUN brew update && brew install golangci-lint bash-completion

RUN echo 'if [ -r "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh" ]; then . "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh"' >>~/.bash_profile

RUN ln -s ~/workspace/ddev/.gotmp/bin/linux_amd64/ddev /usr/local/bin/ddev

# Install custom tools, runtimes, etc.
# For example "bastet", a command-line tetris clone:
# RUN brew install bastet
#
# More information: https://www.gitpod.io/docs/config-docker/
