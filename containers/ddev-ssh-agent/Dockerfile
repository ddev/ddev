# syntax=docker/dockerfile:1
# check=skip=SecretsUsedInArgOrEnv

FROM debian:trixie-slim AS ddev-ssh-agent

# tzdata-legacy - to avoid issues with deprecated timezones
RUN apt-get update && apt-get install -y bash expect file gpg openssh-client socat psmisc tzdata-legacy && apt-get autoclean

# Copy container files
COPY files /

# Setup environment variables; export SSH_AUTH_SOCK from socket directory
ENV SSH_KEY_DIR=/tmp/.ssh
ENV SOCKET_DIR=/tmp/.ssh-agent
ENV SSH_AUTH_SOCK=${SOCKET_DIR}/socket
ENV SSH_AUTH_PROXY_SOCK=${SOCKET_DIR}/proxy-socket

RUN <<EOF
set -eu -o pipefail
chmod a+x /entry.sh
ln -s $SSH_KEY_DIR /home/.ssh
mkdir ${SOCKET_DIR} && mkdir ${SSH_KEY_DIR} && chmod 777 ${SOCKET_DIR} ${SSH_KEY_DIR}
# Don't show "I have no name!" in `docker exec`
# when running as a user without a name in /etc/passwd
cat >> /etc/bash.bashrc << 'BASHRC'
getent passwd "$(id -u)" >/dev/null 2>&1 || PS1='${debian_chroot:+($debian_chroot)}\h:\w\$ '
BASHRC
EOF

HEALTHCHECK --interval=1s --retries=5 --timeout=120s CMD ["/healthcheck.sh"]

VOLUME ${SOCKET_DIR}

ENTRYPOINT ["/entry.sh"]

CMD ["ssh-agent"]
