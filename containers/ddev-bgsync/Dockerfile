FROM alpine:3.8

RUN apk add --no-cache bash sudo rsync file

# Install Unison from source with inotify support + remove compilation tools
ARG UNISON_VERSION=2.51.2
RUN apk add --no-cache --virtual .build-dependencies build-base curl && \
    apk add --no-cache inotify-tools tzdata && \
    apk add --no-cache --repository http://dl-4.alpinelinux.org/alpine/edge/testing/ ocaml && \
    curl -Ll https://github.com/bcpierce00/unison/archive/v${UNISON_VERSION}.tar.gz | tar zxv -C /tmp && \
    cd /tmp/unison-${UNISON_VERSION} && \
    sed -i -e 's/GLIBC_SUPPORT_INOTIFY 0/GLIBC_SUPPORT_INOTIFY 1/' src/fsmonitor/linux/inotify_stubs.c && \
    make UISTYLE=text NATIVE=true STATIC=true && \
    cp src/unison src/unison-fsmonitor /usr/local/bin && \
    apk del .build-dependencies ocaml && \
    rm -rf /tmp/unison-${UNISON_VERSION}

ENV HOME="/root" \
    UNISONLOCALHOSTNAME="container"

# If run as UNISON_USER other than root, it still uses /root as $HOME
RUN mkdir -p $HOME/.unison && chmod ugo+rwx $HOME && chmod ugo+rwx $HOME/.unison

ADD files /

# Copy the bg-sync script into /usr/local/bin.
COPY /files/sync.sh /usr/local/bin/bg-sync
RUN chmod +x /usr/local/bin/bg-sync

HEALTHCHECK --start-period=30s --interval=10s --retries=5 CMD ["/healthcheck.sh"]
CMD ["bg-sync"]
