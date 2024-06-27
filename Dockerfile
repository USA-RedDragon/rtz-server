FROM scratch
COPY hack/passwd /etc/passwd
COPY hack/group /etc/group
COPY connect-server /
USER 65534:65534
ENTRYPOINT [ "/connect-server" ]