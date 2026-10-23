FROM alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
RUN apk add --no-cache ca-certificates sqlite
# COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY hack/passwd /etc/passwd
COPY hack/group /etc/group
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/rtz-server /
USER 65534:65534
ENTRYPOINT [ "/rtz-server" ]