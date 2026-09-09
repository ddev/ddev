FROM debian:trixie-slim AS test-ssh-server

RUN apt-get -qq update && \
    DEBIAN_FRONTEND=noninteractive apt-get -qq install -y -o Dpkg::Options::="--force-confold" --no-install-recommends --no-install-suggests bash openssh-server vim-tiny && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/*

COPY files /

RUN mkdir -p /run/sshd && chmod 755 /run/sshd && \
    ssh-keygen -A && \
    chmod -R go-rwx /root/.ssh

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D", "-e"]
