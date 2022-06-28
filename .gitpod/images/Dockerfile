FROM gitpod/workspace-base as workspace-base
SHELL ["/bin/bash", "-c"]

USER root

RUN curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -

RUN apt-get update >/dev/null && sudo apt-get install -y aspell autojump file mysql-client netcat nodejs python3-pip telnet xdg-utils >/dev/null

RUN pip3 install mkdocs pyspelling pymdown-extensions
RUN npm install -g markdownlint-cli
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin v1.46.2

RUN rm -rf /usr/local/go && curl -sL -o /tmp/go.tar.gz https://go.dev/dl/go1.17.8.linux-amd64.tar.gz && tar -C /usr/local -xzf /tmp/go.tar.gz && rm /tmp/go.tar.gz && ln -s /usr/local/go/bin/go /usr/local/bin/go

USER gitpod

RUN curl -sL -o /tmp/install_ddev.sh https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev.sh && bash /tmp/install_ddev.sh

RUN echo 'if [ -r "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh" ]; then . "/home/linuxbrew/.linuxbrew/etc/profile.d/bash_completion.sh"; fi' >>~/.bashrc

RUN echo 'export PATH=~/bin:$PATH' >>~/.bashrc && mkdir -p ~/bin
RUN echo ". /usr/share/autojump/autojump.sh" >> ~/.bashrc
RUN ln -sf /workspace/ddev/.gotmp/bin/linux_amd64/ddev ~/bin/ddev
RUN mkdir -p ~/.ddev && echo "omit_containers: [ddev-router,ddev-ssh-agent]" >> ~/.ddev/global_config.yaml
RUN sudo apt-get clean && sudo rm -rf /var/lib/apt/lists/*

# a gcc instance named gcc-5 is required for some vscode installations
RUN sudo ln -sf $(which gcc) /usr/local/bin/gcc-5

RUN for item in golang.org/x/tools/gopls@latest github.com/go-delve/delve/cmd/dlv@latest; do \
        go install $item; \
    done
RUN cp ~/go/bin/dlv ~/go/bin/dlv-dap

RUN pip3 install -r docs/mkdocs-pip-requirements
