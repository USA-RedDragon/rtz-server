FROM alpine:3.20
RUN apk add --no-cache ca-certificates sqlite
# COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY hack/passwd /etc/passwd
COPY hack/group /etc/group
COPY rtz-server /
USER 65534:65534
ENTRYPOINT [ "/rtz-server" ]