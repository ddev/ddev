FROM mcr.microsoft.com/devcontainers/base:dev-ubuntu22.04 AS ddev-devcontainers-base

SHELL ["/bin/bash", "-c"]
USER root

RUN curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg
RUN echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_20.x nodistro main" > /etc/apt/sources.list.d/nodesource.list

RUN install -m 0755 -d /etc/apt/keyrings
#RUN curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null
#RUN echo "deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *" | tee /etc/apt/sources.list.d/ddev.list

RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
    $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null


RUN curl -o /usr/bin/slirp4netns -fsSL https://github.com/rootless-containers/slirp4netns/releases/download/v1.1.12/slirp4netns-$(uname -m) \
    && chmod +x /usr/bin/slirp4netns

RUN apt-get update >/dev/null
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y -o Dpkg::Options::="--force-confold" \
    aspell \
    autojump \
    bash-completion \
    build-essential \
    clang \
    docker-ce \
    docker-ce-cli \
    containerd.io \
    file \
    fish \
    git \
    gopls \
    htop \
    iputils-ping \
    jq \
    less \
    lsof \
    man-db \
    mariadb-client \
    nano \
    netcat-traditional \
    nodejs \
    python3-pip \
    software-properties-common \
    ssl-cert \
    sudo \
    telnet \
    time \
    unzip \
    vim \
    zip \
    zsh

# When we need mkdocs we can get it in here better.
#RUN pip3 install mkdocs pyspelling pymdown-extensions
RUN npm install -g markdownlint-cli
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin

RUN rm -rf /usr/local/go && curl -sL -o /tmp/go.tar.gz https://go.dev/dl/$(curl -fsSL "https://go.dev/dl/?mode=json" | jq -r ".[0].version").linux-amd64.tar.gz && tar -C /usr/local -xzf /tmp/go.tar.gz && rm /tmp/go.tar.gz && ln -s /usr/local/go/bin/go /usr/local/bin/go

RUN echo 'export PATH=~/bin:$PATH' >>~/.bashrc && mkdir -p ~/bin
RUN echo ". /usr/share/autojump/autojump.sh" >> ~/.bashrc
RUN ln -sf /workspace/ddev/.gotmp/bin/linux_amd64/ddev ~/bin/ddev
# Use a non-volatile global config location
ENV XDG_CONFIG_HOME=/workspace/.config
RUN mkdir -p ${XDG_CONFIG_HOME}/ddev && echo "omit_containers: [ddev-router]" >> ${XDG_CONFIG_HOME}/ddev/global_config.yaml
# Make a link to the previous global config location,
# in case someone has hardcoded it in their projects somewhere, e.g. in hooks
RUN ln -sf ${XDG_CONFIG_HOME}/ddev ~/.ddev
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

# a gcc instance named gcc-5 is required for some vscode installations
RUN ln -sf $(which gcc) /usr/local/bin/gcc-5

RUN useradd -l -u 33333 -G sudo -md /home/ddevuser -s /bin/bash -p ddevuser ddevuser \
    # Remove `use_pty` option and enable passwordless sudo for users in the 'sudo' group
    && sed -i.bkp -e '/Defaults\tuse_pty/d' -e 's/%sudo\s\+ALL=(ALL\(:ALL\)\?)\s\+ALL/%sudo ALL=NOPASSWD:ALL/g' /etc/sudoers \
    && mkdir -p /workspace && chown -hR ddevuser:ddevuser /workspace \
    && sudo usermod -aG docker ddevuser

ENV HOME=/home/ddevuser
WORKDIR $HOME
# custom Bash prompt
RUN { echo && echo "PS1='\[\033[01;32m\]\u\[\033[00m\] \[\033[01;34m\]\w\[\033[00m\]\$(__git_ps1 \" (%s)\") $ '" ; } >> .bashrc

RUN printf "[safe]\n         	directory = * default.gitconfig /etc/gitconfig" >/etc/gitconfig
RUN cp /etc/gitconfig /home/ddevuser/.gitconfig

# Some ~/.cache and related have been created by root
RUN chown -R ddevuser ~

USER ddevuser

ENV CGO_ENABLED=0
# Go stuff can be added after everything else starts to work.
#RUN for item in golang.org/x/tools/gopls@latest github.com/go-delve/delve/cmd/dlv@latest; do \
#        go install $item; \
#    done && go clean -modcache
#RUN cp ~/go/bin/dlv ~/go/bin/dlv-dap
