FROM alpine:3.23@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40
RUN apk add --no-cache ca-certificates sqlite
# COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY hack/passwd /etc/passwd
COPY hack/group /etc/group
COPY rtz-server /
USER 65534:65534
ENTRYPOINT [ "/rtz-server" ]