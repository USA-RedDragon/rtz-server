FROM scratch

COPY connect-server /

ENTRYPOINT [ "/connect-server" ]