FROM nginx:1.18

ENV MKCERT_VERSION=v1.4.1

ENV DEBIAN_FRONTEND noninteractive
ENV DOCKER_GEN_VERSION 0.7.4
ENV DOCKER_HOST unix:///tmp/docker.sock

# TARGETPLATFORM is Docker buildx's target platform (e.g. linux/arm64), while 
# BUILDPLATFORM is the platform of the build host (e.g. linux/amd64)
ARG TARGETPLATFORM
ARG BUILDPLATFORM

SHELL ["/bin/bash", "-c"]

RUN mkdir /tmp/ddev && \
    if [[ $TARGETPLATFORM == "linux/arm64" ]]; then \
        printf '# jwilder/forego doesnt have an arm64 download so we resort to the original forego repo \n\
FORGO_IS_TGZ="true" \n\
FORGO_FILENAME="forego-stable-linux-arm64.tgz" \n\
FORGO_URL="https://bin.equinox.io/c/ekMN3bCZFUn/forego-stable-linux-arm64.tgz" \n\
# jwilder/docker-gen doesnt have an arm64 download so we created a custom build \n\
DOCKER_GEN_URL="https://github.com/dennisameling/docker-gen/releases/download/0.7.5/docker-gen-linux-arm64-0.7.5.tar.gz" \n\
DOCKER_GEN_FILE="docker-gen-linux-arm64-0.7.5.tar.gz" \n\
# PR for official arm64 support pending: https://github.com/FiloSottile/mkcert/pull/284 \n\
MKCERT_URL="https://github.com/dennisameling/mkcert/releases/download/${MKCERT_VERSION}-ddev/mkcert-${MKCERT_VERSION}-linux-arm64"' > /tmp/ddev/vars; \

    else \
        printf 'FORGO_IS_TGZ="false" \n\
FORGO_URL="https://github.com/jwilder/forego/releases/download/v0.16.1/forego" \n\
DOCKER_GEN_URL="https://github.com/jwilder/docker-gen/releases/download/${DOCKER_GEN_VERSION}/docker-gen-linux-amd64-${DOCKER_GEN_VERSION}.tar.gz" \n\
DOCKER_GEN_FILE="docker-gen-linux-amd64-${DOCKER_GEN_VERSION}.tar.gz" \n\
MKCERT_URL="https://github.com/FiloSottile/mkcert/releases/download/$MKCERT_VERSION/mkcert-${MKCERT_VERSION}-linux-amd64"' > /tmp/ddev/vars; \
    fi

RUN apt-get -qq update && \
    apt-get -qq install --no-install-recommends --no-install-suggests -y \
        ca-certificates certbot curl iputils-ping less python-certbot-nginx procps telnet vim wget && \
    apt-get autoremove -y && \
    apt-get clean -y && \
	rm -rf /var/lib/apt/lists/*

RUN source /tmp/ddev/vars && \
    if [[ $FORGO_IS_TGZ == "true" ]]; then \
        wget -q $FORGO_URL && \
        tar -C /usr/local/bin -xvzf $FORGO_FILENAME && \
        rm /$FORGO_FILENAME && \
        chmod ugo+x /usr/local/bin/forego; \
    else \
        curl -sSL $FORGO_URL -o /usr/local/bin/forego && \
        chmod ugo+x /usr/local/bin/forego; \
    fi

RUN source /tmp/ddev/vars && \
    wget -q $DOCKER_GEN_URL && \
    tar -C /usr/local/bin -xvzf $DOCKER_GEN_FILE && \
    rm /$DOCKER_GEN_FILE
    
RUN mkdir -p /etc/nginx/certs /mnt/ddev-global-cache/mkcert

RUN source /tmp/ddev/vars && \
    curl -sSL $MKCERT_URL -o /usr/local/bin/mkcert && chmod +x /usr/local/bin/mkcert && mkdir -p /root/.local/share && ln -s /mnt/ddev-global-cache/mkcert /root/.local/share/mkcert && mkcert -install

# Configure Nginx and apply fix for very long server names
RUN echo "daemon off;" >> /etc/nginx/nginx.conf \
 && sed -i 's/worker_processes  1/worker_processes  auto/' /etc/nginx/nginx.conf

ADD . /app/
ADD etc /etc
RUN chmod ugo+x /app/healthcheck.sh

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["forego", "start", "-r"]
WORKDIR /app/

HEALTHCHECK --interval=1s --retries=10 --timeout=120s --start-period=10s CMD /app/healthcheck.sh
