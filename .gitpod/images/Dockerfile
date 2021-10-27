FROM gitpod/workspace-full
SHELL ["/bin/bash", "-c"]
ENV HOMEBREW_INSTALL_FROM_API=true

RUN sudo apt-get update >/dev/null && sudo apt-get install -y file mysql-client netcat telnet >/dev/null
RUN brew install bash-completion bats-core drud/ddev/ddev golangci-lint
RUN npm install -g markdownlint-cli

RUN echo 'if [ -r "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh" ]; then . "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh"; fi' >>~/.bashrc

RUN echo 'export PATH=~/bin:$PATH' >>~/.bashrc && mkdir -p ~/bin
RUN ln -sf /workspace/ddev/.gotmp/bin/linux_amd64/ddev ~/bin/ddev

COPY gitpod-setup-ddev.sh /usr/local/bin
