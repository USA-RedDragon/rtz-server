package apimodels

type InboundRPCCall struct {
	ID             int    `json:"id"`
	Method         string `json:"method"`
	JSONRPCVersion string `json:"jsonrpc"`
	Params         any    `json:"params"`
}

type RPCCall struct {
	ID             string `json:"id"`
	Method         string `json:"method"`
	JSONRPCVersion string `json:"jsonrpc"`
	Params         any    `json:"params"`
}

type RPCResponse struct {
	ID             string `json:"id"`
	JSONRPCVersion string `json:"jsonrpc"`
	Result         any    `json:"result"`
	Error          string `json:"error"`
}
