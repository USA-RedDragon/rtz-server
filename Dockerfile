FROM scratch
COPY hack/passwd /etc/passwd
COPY connect-server /
USER 65534
ENTRYPOINT [ "/connect-server" ]