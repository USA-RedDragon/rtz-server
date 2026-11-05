package websocket

import (
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/websocket"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/puzpuzpuz/xsync/v4"
)

// unsubscriber is the part of *nats.Subscription that a dongle uses.
type unsubscriber interface {
	Unsubscribe() error
}

// dongle is one device connection. Nothing in it is closed while the
// connection may still use it; senders watch done instead.
type dongle struct {
	rpc     *RPCWebsocket
	device  *models.Device
	metrics *metrics.Metrics
	conn    *gorillaWebsocket.Conn
	writer  websocket.Writer
	// done is closed when the connection ends.
	done <-chan struct{}
	// closed is closed once OnDisconnect has finished.
	closed  chan struct{}
	pending *xsync.MapOf[string, chan apimodels.RPCResponse]
	natsSub unsubscriber
}
