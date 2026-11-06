FROM alpine
RUN echo built-at-$(date +%s) > /b.txt
