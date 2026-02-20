FROM alpine:latest
LABEL org.opencontainers.image.source=https://github.com/pteich/kcplens

COPY kcplens /usr/bin/kcplens

ENTRYPOINT ["/usr/bin/kcplens"]