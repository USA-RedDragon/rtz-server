FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
RUN apk add --no-cache ca-certificates sqlite
# COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY hack/passwd /etc/passwd
COPY hack/group /etc/group
COPY rtz-server /
USER 65534:65534
ENTRYPOINT [ "/rtz-server" ]