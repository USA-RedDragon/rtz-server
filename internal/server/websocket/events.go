package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/connect-server/internal/events"
	"github.com/USA-RedDragon/connect-server/internal/websocket"
	gorillaWebsocket "github.com/gorilla/websocket"
)

type EventsWebsocket struct {
	websocket.Websocket
	cancel           context.CancelFunc
	websocketChannel chan events.Event
	eventsChannel    chan events.Event
	connectedCount   uint
}

func CreateEventsWebsocket(eventsChannel chan events.Event) *EventsWebsocket {
	ew := &EventsWebsocket{
		websocketChannel: make(chan events.Event),
		eventsChannel:    eventsChannel,
	}

	go ew.start()

	return ew
}

func (c *EventsWebsocket) start() {
	for {
		event := <-c.eventsChannel
		// If the websocket is closed, we just want to drop the event
		if c.connectedCount > 0 {
			c.websocketChannel <- event
		}
	}
}

func (c *EventsWebsocket) OnMessage(_ context.Context, _ *http.Request, _ websocket.Writer, msg []byte, msgType int) {
	slog.Info("Received message:", "message", string(msg), "type", msgType)
}

func (c *EventsWebsocket) OnConnect(ctx context.Context, _ *http.Request, w websocket.Writer) {
	newCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	slog.Info("New websocket connection")

	c.connectedCount++

	go func() {
		channel := make(chan websocket.Message)
		for {
			select {
			case <-ctx.Done():
				return
			case <-newCtx.Done():
				return
			case event := <-c.websocketChannel:
				if event == nil {
					return
				}
				switch event.GetType() {
				case events.EventTypeTest:
					_, ok := event.(events.TestEvent)
					if !ok {
						slog.Warn("Error casting event to TestEvent")
						continue
					}
				}
				eventDataJSON, err := json.Marshal(event)
				if err != nil {
					slog.Warn("Error marshalling event data:", err)
					continue
				}
				w.WriteMessage(websocket.Message{
					Type: gorillaWebsocket.TextMessage,
					Data: eventDataJSON,
				})
			case <-channel:
				// We don't actually want to receive messages from the client
				continue
			}
		}
	}()
}

func (c *EventsWebsocket) OnDisconnect(ctx context.Context, _ *http.Request) {
	slog.Info("Websocket disconnected")
	c.connectedCount--
	c.cancel()
}
