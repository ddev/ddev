FROM traefik:v3 AS ddev-traefik-router

ENV TRAEFIK_MONITOR_PORT=10999
# socat serves the informative "no matching project" fallback page as an
# actual HTTP 404 response, see docker-entrypoint.sh and unmatched-route.html
RUN apk add --no-cache bash curl file htop jq openssl socat vim yq
WORKDIR /mnt/ddev-global-cache/traefik
ADD monitor-traefik-stderr.sh /usr/local/bin/monitor-traefik-stderr.sh
RUN chmod ugo+rx /usr/local/bin/monitor-traefik-stderr.sh
ADD docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod ugo+rx /usr/local/bin/docker-entrypoint.sh
ADD ddev-router-fallback-responder.sh /usr/local/bin/ddev-router-fallback-responder.sh
RUN chmod ugo+rx /usr/local/bin/ddev-router-fallback-responder.sh
RUN mkdir -p /var/www/ddev-router-fallback
ADD unmatched-route.html /var/www/ddev-router-fallback/unmatched-route.html
# Make Traefik commands work without --configFile by using default location
# https://doc.traefik.io/traefik/getting-started/configuration-overview/#configuration-file
RUN mkdir -p /etc/traefik && ln -s /mnt/ddev-global-cache/traefik/.static_config.yaml /etc/traefik/traefik.yaml
COPY /traefik_healthcheck.sh /healthcheck.sh
RUN chmod ugo+rx /healthcheck.sh
HEALTHCHECK --interval=1s --timeout=120s --retries=1 --start-period=120s CMD /healthcheck.sh
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh", "/usr/local/bin/traefik"]
