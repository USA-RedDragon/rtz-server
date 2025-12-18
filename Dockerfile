FROM alpine:3.23@sha256:be171b562d67532ea8b3c9d1fc0904288818bb36fc8359f954a7b7f1f9130fb2
RUN apk add --no-cache ca-certificates sqlite
# COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY hack/passwd /etc/passwd
COPY hack/group /etc/group
COPY rtz-server /
USER 65534:65534
ENTRYPOINT [ "/rtz-server" ]