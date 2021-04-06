FROM gitpod/workspace-full
SHELL ["/bin/bash", "-c"]

RUN apt-get update && apt-get install netcat telnet
RUN brew update && brew install bash-completion drud/ddev/ddev golangci-lint

RUN echo 'if [ -r "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh" ]; then . "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh"; fi' >>~/.bashrc

RUN echo 'export PATH=~/bin:$PATH' >>~/.bashrc && mkdir -p ~/bin
RUN ln -sf /workspace/ddev/.gotmp/bin/linux_amd64/ddev ~/bin/ddev

RUN curl -o ~/bin/gitpod-setup-ddev.sh --fail -lLs https://raw.githubusercontent.com/shaal/ddev-gitpod/main/.ddev/gitpod-setup-ddev.sh && chmod +x ~/bin/gitpod-setup-ddev.sh
RUN mkcert -install

# Install custom tools, runtimes, etc.
# For example "bastet", a command-line tetris clone:
# RUN brew install bastet
#
# More information: https://www.gitpod.io/docs/config-docker/
