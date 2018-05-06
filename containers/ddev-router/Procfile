nginx: nginx
dockergen: docker-gen -watch -only-exposed -notify "chmod ugo+x /gen-cert.sh && /gen-cert.sh" /app/gen-cert.sh.tmpl /gen-cert.sh
dockergen: docker-gen -watch -only-exposed -notify "sleep 1 && nginx -s reload" /app/nginx.tmpl /etc/nginx/conf.d/default.conf
