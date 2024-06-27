export function getWebsocketURI() {
  const loc = window.location;
  let newURI;
  if (loc.protocol === 'https:') {
    newURI = 'wss:';
  } else {
    newURI = 'ws:';
  }
  newURI += '//' + loc.host + '/ws';
  console.log('Websocket URI: "' + newURI + '"');
  return newURI;
}