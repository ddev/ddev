FROM nginx:1.15.12

ENV MKCERT_VERSION=v1.3.0

ENV DEBIAN_FRONTEND noninteractive
ENV DOCKER_GEN_VERSION 0.7.4
ENV DOCKER_HOST unix:///tmp/docker.sock

RUN apt-get -qq update && \
    apt-get -qq install --no-install-recommends --no-install-suggests -y \
        ca-certificates wget vim less telnet curl procps && \
    apt-get autoremove -y && \
    apt-get clean -y && \
	rm -rf /var/lib/apt/lists/*

ADD https://github.com/jwilder/forego/releases/download/v0.16.1/forego /usr/local/bin/forego

RUN wget -q https://github.com/jwilder/docker-gen/releases/download/$DOCKER_GEN_VERSION/docker-gen-linux-amd64-$DOCKER_GEN_VERSION.tar.gz && \
    tar -C /usr/local/bin -xvzf docker-gen-linux-amd64-$DOCKER_GEN_VERSION.tar.gz && \
    rm /docker-gen-linux-amd64-$DOCKER_GEN_VERSION.tar.gz && \
    chmod ugo+x /usr/local/bin/forego && \
    mkdir -p /etc/nginx/certs /mnt/ddev-global-cache/mkcert

RUN curl -sSL https://github.com/FiloSottile/mkcert/releases/download/$MKCERT_VERSION/mkcert-$MKCERT_VERSION-linux-amd64 -o /usr/local/bin/mkcert && chmod +x /usr/local/bin/mkcert && mkdir -p /root/.local/share && ln -s /mnt/ddev-global-cache/mkcert /root/.local/share/mkcert && mkcert -install

# Configure Nginx and apply fix for very long server names
RUN echo "daemon off;" >> /etc/nginx/nginx.conf \
 && sed -i 's/worker_processes  1/worker_processes  auto/' /etc/nginx/nginx.conf

ADD . /app/
RUN chmod ugo+x /app/healthcheck.sh

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["forego", "start", "-r"]
WORKDIR /app/

HEALTHCHECK --interval=5s --retries=5 CMD /app/healthcheck.sh
