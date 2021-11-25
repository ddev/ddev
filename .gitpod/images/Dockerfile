FROM gitpod/workspace-full as workspace-base
SHELL ["/bin/bash", "-c"]
ENV HOMEBREW_INSTALL_FROM_API=true

RUN sudo apt-get update >/dev/null && sudo apt-get install -y autojump file mysql-client netcat telnet >/dev/null
RUN brew install bash-completion bats-core drud/ddev/ddev golang golangci-lint
RUN npm install -g markdownlint-cli

RUN echo 'if [ -r "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh" ]; then . "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh"; fi' >>~/.bashrc

RUN echo 'export PATH=~/bin:$PATH' >>~/.bashrc && mkdir -p ~/bin
RUN echo ". /usr/share/autojump/autojump.sh" >> ~/.bashrc
RUN ln -sf /workspace/ddev/.gotmp/bin/linux_amd64/ddev ~/bin/ddev
RUN ddev config global --omit-containers=ddev-router,dba,ddev-ssh-agent
RUN brew cleanup
RUN sudo apt-get clean && sudo rm -rf /var/lib/apt/lists/*

COPY gitpod-setup-ddev.sh /usr/local/bin
# End workspace-base

FROM scratch as ddev-gitpod-base
SHELL ["/bin/bash", "-c"]
ENV HOMEBREW_INSTALL_FROM_API=true
COPY --from=workspace-base / /

