FROM phpmyadmin/phpmyadmin:4.9.2-1

RUN apk add --no-cache curl curl-dev bash vim

HEALTHCHECK --interval=5s --retries=5 CMD curl -s --fail http://127.0.0.1  >/dev/null || exit 1

