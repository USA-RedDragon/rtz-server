package websocket

type Message struct {
	Type int
	Data []byte
}

type Writer interface {
	WriteMessage(message Message)
	Error(message string)
}

type wsWriter struct {
	writer chan Message
	error  chan string
}

func (w wsWriter) WriteMessage(message Message) {
	w.writer <- message
}

func (w wsWriter) Error(message string) {
	w.error <- message
}
