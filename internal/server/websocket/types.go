package websocket

import (
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/utils"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

type bidiChannel struct {
	open     bool
	inbound  chan apimodels.RPCCall
	outbound chan apimodels.RPCResponse
}

type dongle struct {
	bidiChannel    *bidiChannel
	channelWatcher *utils.ChannelWatcher[apimodels.RPCResponse]
	conn           *gorillaWebsocket.Conn
	natsSub        *nats.Subscription
}
