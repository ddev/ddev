FROM xhgui/xhgui:0.21 AS xhgui-docker

FROM scratch AS ddev-xhgui

### --------------------------------xhgui-docker----------------------------------
### XHGui Docker image creates a volume for /run/nginx, which we want to remove.
### We copy all files from xhgui/xhgui, and set ENV, WORKDIR, CMD, EXPOSE.
### Source https://github.com/perftools/xhgui/blob/0.21.x/Dockerfile
COPY --from=xhgui-docker / /
ENV PHP_INI_DIR=/etc/php7
ARG APPDIR=/var/www/xhgui
ARG WEBROOT=$APPDIR/webroot
WORKDIR $APPDIR
RUN mkdir -p /run/nginx
CMD ["sh", "-c", "nginx && exec php-fpm"]
EXPOSE 80
### ------------------------------END xhgui-docker--------------------------------

RUN apk add bash curl
ADD /var /var
ADD /etc /etc
RUN echo 'memory_limit=512M' >> $PHP_INI_DIR/conf.d/99-memory-limit.ini
