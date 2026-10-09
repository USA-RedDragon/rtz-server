package websocket

import "time"

const (
	writeWait        = 10 * time.Second
	errorKey         = "error"
	msgTryAgainLater = "Try again later"
)
