FROM xhgui/xhgui:0.23 AS ddev-xhgui

RUN apk add --no-cache bash curl tzdata
ADD /var /var
ADD /etc /etc
RUN echo 'memory_limit=512M' >> $PHP_INI_DIR/conf.d/99-memory-limit.ini
RUN echo 'post_max_size=100M' >> $PHP_INI_DIR/conf.d/99-post-max-size.ini
ADD healthcheck.sh /healthcheck.sh
RUN chmod ugo+x /healthcheck.sh
HEALTHCHECK --interval=1s --retries=5 --timeout=120s CMD ["/healthcheck.sh"]
CMD ["sh", "-c", "rm -f /tmp/healthy; nginx -g 'pid /dev/shm/nginx.pid;' && exec php-fpm"]
# ddev-release-marker: v1.25.5
