FROM xhgui/xhgui:0.23 AS ddev-xhgui

RUN apk add --no-cache bash curl tzdata
ADD /var /var
ADD /etc /etc
RUN echo 'memory_limit=512M' >> $PHP_INI_DIR/conf.d/99-memory-limit.ini
ADD healthcheck.sh /healthcheck.sh
RUN chmod ugo+x /healthcheck.sh
HEALTHCHECK --interval=1s --retries=5 --timeout=120s CMD ["/healthcheck.sh"]
