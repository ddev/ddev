FROM traefik:3 AS ddev-traefik-router

ENV TRAEFIK_MONITOR_PORT=10999
RUN apk add --no-cache bash curl file htop jq openssl vim yq
WORKDIR /mnt/ddev-global-cache/traefik
ADD monitor-traefik-stderr.sh /usr/local/bin/monitor-traefik-stderr.sh
RUN chmod ugo+rx /usr/local/bin/monitor-traefik-stderr.sh
# Make Traefik commands work without --configFile by using default location
# https://doc.traefik.io/traefik/getting-started/configuration-overview/#configuration-file
RUN mkdir -p /etc/traefik && ln -s /mnt/ddev-global-cache/traefik/.static_config.yaml /etc/traefik/traefik.yaml
COPY /traefik_healthcheck.sh /healthcheck.sh
RUN chmod ugo+rx /healthcheck.sh
HEALTHCHECK --interval=1s --timeout=120s --retries=1 --start-period=120s CMD /healthcheck.sh
ENTRYPOINT ["/usr/local/bin/monitor-traefik-stderr.sh", "/usr/local/bin/traefik"]
