FROM alpine:3.20
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY hack/passwd /etc/passwd
COPY hack/group /etc/group
COPY connect-server /
USER 65534:65534
ENTRYPOINT [ "/connect-server" ]