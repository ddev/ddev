FROM traefik:2.10

RUN apk add bash curl htop vim
WORKDIR /mnt/ddev-global-cache/traefik
COPY /traefik_healthcheck.sh /healthcheck.sh
RUN chmod ugo+rx /healthcheck.sh

