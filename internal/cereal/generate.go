package cereal

//go:generate sh download.sh
//go:generate go install capnproto.org/go/capnp/v3/capnpc-go@latest
//go:generate sh -c "capnp compile -I $(go list -m -f '{{.Dir}}' capnproto.org/go/capnp/v3)/std -ogo *.capnp"
