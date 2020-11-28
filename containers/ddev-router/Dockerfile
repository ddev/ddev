FROM nginx:1.18

ENV MKCERT_VERSION=v1.4.6

ENV DEBIAN_FRONTEND noninteractive
ENV DOCKER_GEN_VERSION 0.7.7
ENV DOCKER_HOST unix:///tmp/docker.sock

ARG TARGETARCH

SHELL ["/bin/bash", "-c"]

# Get forego, which may be either a binary download (jwilder) or
# a tarball (from https://github.com/ddollar/forego download)
RUN set -eu -o pipefail && \
    FOREGO_URL="https://github.com/jwilder/forego/releases/download/v0.16.1/forego" && \
    if [[ "${TARGETARCH}" = "arm"* ]]; then \
        FOREGO_URL="https://bin.equinox.io/c/ekMN3bCZFUn/forego-stable-linux-${TARGETARCH}.tgz"; \
    fi && \
    cd /tmp && curl -sSL -O "${FOREGO_URL}"  && \
    if [ ${FOREGO_URL##*.} = "tgz" ]; then \
        tar -zxf ${FOREGO_URL##*/} ; \
    fi && \
    mv forego /usr/local/bin && chmod +x /usr/local/bin/forego

RUN apt-get -qq update && \
    apt-get -qq install --no-install-recommends --no-install-suggests -y \
        ca-certificates certbot curl iputils-ping less python-certbot-nginx procps telnet vim wget && \
    apt-get autoremove -y && \
    apt-get clean -y && \
	rm -rf /var/lib/apt/lists/*

# Download docker-gen
RUN set -eu -o pipefail && \
    export DOCKER_GEN_FILE="docker-gen-linux-${TARGETARCH}-${DOCKER_GEN_VERSION}.tar.gz" && \
    cd /tmp && wget -q https://github.com/drud/docker-gen/releases/download/${DOCKER_GEN_VERSION}/${DOCKER_GEN_FILE} && \
    tar -C /usr/local/bin -xzf "${DOCKER_GEN_FILE}" && \
    rm $DOCKER_GEN_FILE

RUN mkdir -p /etc/nginx/certs /mnt/ddev-global-cache/mkcert

# curl mkcert and install it
RUN set -eu -o pipefail && \
    curl -sSL https://github.com/drud/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-${TARGETARCH} -o /usr/local/bin/mkcert && \
    chmod +x /usr/local/bin/mkcert && \
    mkdir -p /root/.local/share && \
    ln -s /mnt/ddev-global-cache/mkcert /root/.local/share/mkcert && \
    mkcert -install


# Configure Nginx and apply fix for very long server names
RUN echo "daemon off;" >> /etc/nginx/nginx.conf \
 && sed -i 's/worker_processes  1/worker_processes  auto/' /etc/nginx/nginx.conf

# We don't want the default.conf provided by nginx package
RUN rm -f /etc/nginx/conf.d/default.conf

ADD . /app/
ADD etc /etc
RUN chmod ugo+x /app/healthcheck.sh

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["forego", "start", "-r"]
WORKDIR /app/

HEALTHCHECK --interval=1s --retries=10 --timeout=120s --start-period=10s CMD /app/healthcheck.sh
