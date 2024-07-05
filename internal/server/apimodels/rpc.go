package apimodels

type RPCCall struct {
	ID             int            `json:"id"`
	Method         string         `json:"method"`
	JSONRPCVersion string         `json:"jsonrpc"`
	Params         map[string]any `json:"params"`
}
