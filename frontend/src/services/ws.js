export default {
  connect(url, onMessage) {
    const ws = new Websocket(url, onMessage);
    ws.connect();
    return {
      close() {
        ws.socket.close();
      },
    };
  },
};

const initialReconnectDelay = 300;
const maxReconnectDelay = 15000;

class Websocket {
  constructor(url, onMessage) {
    this.url = url;
    this.timeoutTimer = null;
    this.timeout = 3000;
    this.socket = null;
    this.onMessage = onMessage;
    this.currentReconnectDelay = initialReconnectDelay;
  }

  connect() {
    this.socket = new WebSocket(this.url);
    this.mapSocketEvents();
  }

  onWebsocketOpen() {
    console.log('Connected to websocket');
    this.currentReconnectDelay = initialReconnectDelay;
    this.socket.send('PING');
  }

  onWebsocketError() {
    console.log('Disconnected from websocket');
    this.socket = null;
    this.reconnectToWebsocket();
  }

  reconnectToWebsocket() {
    setTimeout(() => {
      if (this.currentReconnectDelay < maxReconnectDelay) {
        this.currentReconnectDelay *= 2;
      }
      this.connect();
    }, this.currentReconnectDelay + Math.floor(Math.random() * 1000));
  }

  mapSocketEvents() {
    this.socket.addEventListener('open', this.onWebsocketOpen.bind(this));
    this.socket.addEventListener('error', this.onWebsocketError.bind(this));

    this.socket.addEventListener('message', (event) => {
      if (event.data == 'PONG') {
        setTimeout(() => {
          this.socket.send('PING');
        }, 1000);
        return;
      } else {
        this.onMessage(event);
      }
    });
  }
}
