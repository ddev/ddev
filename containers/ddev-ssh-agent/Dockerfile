# ddev-ssh-agent, forked from https://github.com/nardeas/docker-ssh-agent
# at https://github.com/nardeas/docker-ssh-agent/commit/fb6822d0003d1c0a795e183f5d257c2540fa74a4

# Docker Image for SSH agent container. Last revision 26.4.2018
#
# Copyright (c) Andreas Urbanski, 2018
#
# Permission is hereby granted, free of charge, to any person obtaining a
# copy of this software and associated documentation files (the "Software"),
# to deal in the Software without restriction, including without limitation
# the rights to use, copy, modify, merge, publish, distribute, sublicense,
# and/or sell copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included
# in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS
# OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
# THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.
FROM alpine:3.8

# Install dependencies
RUN apk add --no-cache \
	bash \
	expect \
	file \
	openssh \
	socat \
	sudo \
	&& rm -rf /var/cache/apk/*

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

HEALTHCHECK --interval=2s --retries=5 CMD ["/healthcheck.sh"]

VOLUME ${SOCKET_DIR}

ENTRYPOINT ["/entry.sh"]

CMD ["ssh-agent"]
