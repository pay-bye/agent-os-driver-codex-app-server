package control

type threadStartParams struct {
	CWD string `json:"cwd"`
}

type turnStartParams struct {
	ThreadID string      `json:"threadId"`
	Input    []userInput `json:"input"`
}

type userInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type initializeParams struct {
	ClientInfo clientInfo `json:"clientInfo"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type rpcRequest struct {
	Version string `json:"jsonrpc"`
	ID      uint64 `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcNotification struct {
	Version string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}
