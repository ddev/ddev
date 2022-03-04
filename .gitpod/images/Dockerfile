FROM gitpod/workspace-base as workspace-base
SHELL ["/bin/bash", "-c"]
ENV HOMEBREW_INSTALL_FROM_API=true

### Homebrew ###
USER gitpod
RUN mkdir ~/.cache && /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
ENV PATH=$PATH:/home/linuxbrew/.linuxbrew/bin:/home/linuxbrew/.linuxbrew/sbin/
ENV MANPATH="$MANPATH:/home/linuxbrew/.linuxbrew/share/man"
ENV INFOPATH="$INFOPATH:/home/linuxbrew/.linuxbrew/share/info"
RUN sudo apt remove -y cmake \
    && brew update && brew install cmake

USER root
RUN curl -o /var/lib/apt/dazzle-marks/docker.gpg -fsSL https://download.docker.com/linux/ubuntu/gpg \
    && apt-key add /var/lib/apt/dazzle-marks/docker.gpg \
    && add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
    && install-packages docker-ce docker-ce-cli containerd.io

RUN curl -o /usr/bin/slirp4netns -fsSL https://github.com/rootless-containers/slirp4netns/releases/download/v1.1.12/slirp4netns-$(uname -m) \
    && chmod +x /usr/bin/slirp4netns

RUN curl -o /usr/local/bin/docker-compose -fsSL https://github.com/docker/compose/releases/download/1.29.2/docker-compose-Linux-x86_64 \
    && chmod +x /usr/local/bin/docker-compose
USER gitpod
RUN sudo apt-get update >/dev/null && sudo apt-get install -y autojump file mysql-client netcat telnet xdg-utils >/dev/null
RUN brew install bash-completion bats-core drud/ddev/ddev golang golangci-lint nodejs
RUN npm install -g markdownlint-cli

RUN echo 'if [ -r "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh" ]; then . "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh"; fi' >>~/.bashrc

RUN echo 'export PATH=~/bin:$PATH' >>~/.bashrc && mkdir -p ~/bin
RUN echo ". /usr/share/autojump/autojump.sh" >> ~/.bashrc
RUN ln -sf /workspace/ddev/.gotmp/bin/linux_amd64/ddev ~/bin/ddev
RUN ddev config global --omit-containers=ddev-router,dba,ddev-ssh-agent
RUN brew cleanup
RUN sudo apt-get clean && sudo rm -rf /var/lib/apt/lists/*

COPY gitpod-setup-ddev.sh /usr/local/bin
# a gcc instance named gcc-5 is required for some vscode installations
RUN sudo ln -sf $(which gcc) /usr/local/bin/gcc-5

RUN for item in golang.org/x/tools/gopls@latest github.com/go-delve/delve/cmd/dlv@latest; do \
        go install $item; \
    done
RUN cp ~/go/bin/dlv ~/go/bin/dlv-dap

# End workspace-base

