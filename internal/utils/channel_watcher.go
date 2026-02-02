package utils

import (
	"github.com/puzpuzpuz/xsync/v4"
)

type ChannelWatcher[T any] struct {
	ch          chan T
	subscribers *xsync.MapOf[string, func(T)]
}

func NewChannelWatcher[T any](ch chan T) *ChannelWatcher[T] {
	return &ChannelWatcher[T]{
		ch:          ch,
		subscribers: xsync.NewMapOf[string, func(T)](),
	}
}

func (cw *ChannelWatcher[T]) WatchChannel(key func(T) string) {
	for {
		response, more := <-cw.ch
		if !more {
			return
		}
		if subscriber, loaded := cw.subscribers.LoadAndDelete(key(response)); loaded {
			subscriber(response)
		}
	}
}

func (cw *ChannelWatcher[T]) Subscribe(id string, subscriber func(T)) {
	cw.subscribers.Store(id, subscriber)
}
