package websocket

import "time"

const (
	writeWait        = 10 * time.Second
	pongWait         = 60 * time.Second
	pingPeriod       = pongWait * 9 / 10
	errorKey         = "error"
	msgTryAgainLater = "Try again later"
)
