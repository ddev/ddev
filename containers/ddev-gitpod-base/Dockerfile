# https://github.com/ddev/ddev/pull/6613 had an attempt to build our own
# multi-arch workspace-base image, but it had its own problems and was abandoned.
FROM gitpod/workspace-base:latest AS ddev-gitpod-base
SHELL ["/bin/bash", "-c"]

USER root

RUN curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg
RUN echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_20.x nodistro main" > /etc/apt/sources.list.d/nodesource.list

RUN install -m 0755 -d /etc/apt/keyrings
RUN curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | sudo tee /etc/apt/keyrings/ddev.gpg > /dev/null
RUN echo "deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *" | sudo tee /etc/apt/sources.list.d/ddev.list

RUN apt-get update >/dev/null && sudo apt-get install -y aspell autojump ddev file mysql-client netcat nodejs python3-pip telnet >/dev/null

RUN pip3 install mkdocs pyspelling pymdown-extensions
RUN npm install -g markdownlint-cli
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin

RUN rm -rf /usr/local/go && curl -sL -o /tmp/go.tar.gz https://go.dev/dl/$(curl -fsSL "https://go.dev/dl/?mode=json" | jq -r ".[0].version").linux-amd64.tar.gz && tar -C /usr/local -xzf /tmp/go.tar.gz && rm /tmp/go.tar.gz && ln -s /usr/local/go/bin/go /usr/local/bin/go

USER gitpod

RUN echo 'if [ -r "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh" ]; then . "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh"; fi' >>~/.bashrc

RUN echo 'export PATH=~/bin:$PATH' >>~/.bashrc && mkdir -p ~/bin
RUN echo ". /usr/share/autojump/autojump.sh" >> ~/.bashrc
RUN ln -sf /workspace/ddev/.gotmp/bin/linux_amd64/ddev ~/bin/ddev
# Use a non-volatile global config location
ENV XDG_CONFIG_HOME=/workspace/.config
RUN mkdir -p ${XDG_CONFIG_HOME}/ddev && echo "omit_containers: [ddev-router]" >> ${XDG_CONFIG_HOME}/ddev/global_config.yaml
# Make a link to the previous global config location,
# in case someone has hardcoded it in their projects somewhere, e.g. in hooks
RUN ln -sf ${XDG_CONFIG_HOME}/ddev ~/.ddev
RUN sudo apt-get clean && sudo rm -rf /var/lib/apt/lists/*

# a gcc instance named gcc-5 is required for some vscode installations
RUN sudo ln -sf $(which gcc) /usr/local/bin/gcc-5

# Some ~/.cache and related have been created by root
RUN sudo chown -R gitpod ~
RUN for item in golang.org/x/tools/gopls@latest github.com/go-delve/delve/cmd/dlv@latest; do \
        go install $item; \
    done && go clean -modcache
RUN cp ~/go/bin/dlv ~/go/bin/dlv-dap

RUN cd /tmp && curl -LO --fail https://raw.githubusercontent.com/ddev/ddev/main/docs/mkdocs-pip-requirements && pip3 install -r /tmp/mkdocs-pip-requirements
