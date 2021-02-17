FROM debian:buster-slim

RUN apt-get update && apt-get install -y bash expect file openssh-client socat psmisc && apt autoclean

# Copy container files
COPY files /
RUN chmod a+x /entry.sh

# Setup environment variables; export SSH_AUTH_SOCK from socket directory
ENV SSH_KEY_DIR /tmp/.ssh
ENV SOCKET_DIR /tmp/.ssh-agent
ENV SSH_AUTH_SOCK ${SOCKET_DIR}/socket
ENV SSH_AUTH_PROXY_SOCK ${SOCKET_DIR}/proxy-socket

RUN ln -s $SSH_KEY_DIR /home/.ssh

RUN mkdir ${SOCKET_DIR} && mkdir ${SSH_KEY_DIR} && chmod 777 ${SOCKET_DIR} ${SSH_KEY_DIR}

HEALTHCHECK --interval=1s --retries=5 --timeout=120s CMD ["/healthcheck.sh"]

VOLUME ${SOCKET_DIR}

ENTRYPOINT ["/entry.sh"]

CMD ["ssh-agent"]
