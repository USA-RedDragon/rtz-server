package websocket

import (
	"context"
	"errors"
)

// ErrConnectionClosed is returned when writing to a connection that has ended.
var ErrConnectionClosed = errors.New("websocket connection closed")

type Message struct {
	Type int
	Data []byte
}

// Writer queues messages to be sent on a connection.
type Writer interface {
	// WriteMessage queues message, waiting while the queue is full. It returns
	// ErrConnectionClosed once the connection has ended, or ctx's error.
	WriteMessage(ctx context.Context, message Message) error
}

type wsWriter struct {
	messages chan Message
	done     <-chan struct{}
}

func (w wsWriter) WriteMessage(ctx context.Context, message Message) error {
	select {
	case <-w.done:
		return ErrConnectionClosed
	default:
	}
	select {
	case w.messages <- message:
		return nil
	case <-w.done:
		return ErrConnectionClosed
	case <-ctx.Done():
		select {
		case <-w.done:
			return ErrConnectionClosed
		default:
			return ctx.Err()
		}
	}
}
