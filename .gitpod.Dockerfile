FROM gitpod/workspace-full
SHELL ["/bin/bash", "-c"]

RUN brew update && brew install bash-completion drud/ddev/ddev golangci-lint

RUN echo 'if [ -r "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh" ]; then . "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh"; fi' >>~/.bash_profile

RUN echo 'export PATH=~/bin:$PATH' >>~/.bash_profile && mkdir -p ~/bin
RUN ln -sf /workspace/ddev/.gotmp/bin/linux_amd64/ddev ~/bin/ddev

RUN mkcert -install

# Install custom tools, runtimes, etc.
# For example "bastet", a command-line tetris clone:
# RUN brew install bastet
#
# More information: https://www.gitpod.io/docs/config-docker/
