FROM xhgui/xhgui:0.23 AS ddev-xhgui

RUN apk add --no-cache bash curl
ADD /var /var
ADD /etc /etc
RUN echo 'memory_limit=512M' >> $PHP_INI_DIR/conf.d/99-memory-limit.ini
