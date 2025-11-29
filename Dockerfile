FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY cafs /usr/local/bin/cafs

ENTRYPOINT ["cafs"]
