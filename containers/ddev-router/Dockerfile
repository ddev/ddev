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

RUN apt-get -qq update && \
    apt-get -qq install --no-install-recommends --no-install-suggests -y \
        ca-certificates certbot curl iputils-ping less python-certbot-nginx procps telnet vim wget && \
    apt-get autoremove -y && \
    apt-get clean -y && \
	rm -rf /var/lib/apt/lists/*

RUN if [[ $TARGETPLATFORM == "linux/arm64" ]]; then \
        # jwilder/forego doesn't have an arm64 download so we resort to the original forego repo
        wget -q https://bin.equinox.io/c/ekMN3bCZFUn/forego-stable-linux-arm64.tgz && \
        tar -C /usr/local/bin -xvzf forego-stable-linux-arm64.tgz && \
        rm /forego-stable-linux-arm64.tgz && \
        chmod ugo+x /usr/local/bin/forego; \
    else \
        curl -sSL https://github.com/jwilder/forego/releases/download/v0.16.1/forego -o /usr/local/bin/forego && \
        chmod ugo+x /usr/local/bin/forego; \
    fi

RUN if [[ $TARGETPLATFORM == "linux/arm64" ]]; then \
        # jwilder/docker-gen doesn't have an arm64 download so we created a custom build
        wget -q https://github.com/dennisameling/docker-gen/releases/download/0.7.5/docker-gen-linux-arm64-0.7.5.tar.gz && \
        tar -C /usr/local/bin -xvzf docker-gen-linux-arm64-0.7.5.tar.gz && \
        rm /docker-gen-linux-arm64-0.7.5.tar.gz; \
    else \
        wget -q https://github.com/jwilder/docker-gen/releases/download/$DOCKER_GEN_VERSION/docker-gen-linux-amd64-$DOCKER_GEN_VERSION.tar.gz && \
        tar -C /usr/local/bin -xvzf docker-gen-linux-amd64-$DOCKER_GEN_VERSION.tar.gz && \
        rm /docker-gen-linux-amd64-$DOCKER_GEN_VERSION.tar.gz; \
    fi
    
RUN mkdir -p /etc/nginx/certs /mnt/ddev-global-cache/mkcert

RUN if [[ $TARGETPLATFORM == "linux/arm64" ]]; then \
        # PR for official arm64 support pending: https://github.com/FiloSottile/mkcert/pull/284
        curl -sSL https://github.com/dennisameling/mkcert/releases/download/$MKCERT_VERSION-ddev/mkcert-$MKCERT_VERSION-linux-arm64 -o /usr/local/bin/mkcert && chmod +x /usr/local/bin/mkcert && mkdir -p /root/.local/share && ln -s /mnt/ddev-global-cache/mkcert /root/.local/share/mkcert && mkcert -install; \
    else \
        curl -sSL https://github.com/FiloSottile/mkcert/releases/download/$MKCERT_VERSION/mkcert-$MKCERT_VERSION-linux-amd64 -o /usr/local/bin/mkcert && chmod +x /usr/local/bin/mkcert && mkdir -p /root/.local/share && ln -s /mnt/ddev-global-cache/mkcert /root/.local/share/mkcert && mkcert -install; \
    fi

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
