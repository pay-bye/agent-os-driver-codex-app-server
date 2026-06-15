package control

import (
	"context"
	"errors"
	"fmt"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
	"net"
	"sync/atomic"
	"time"
)

const (
	TurnCompleted   TurnStatus = "completed"
	TurnFailed      TurnStatus = "failed"
	TurnInProgress  TurnStatus = "inProgress"
	TurnInterrupted TurnStatus = "interrupted"
)

type TurnStatus string

type Turn struct {
	ID     string
	Status TurnStatus
}

func (t Turn) Completed() bool {
	return t.Status == TurnCompleted
}

func (t Turn) Terminal() bool {
	return t.Status == TurnCompleted || t.Status == TurnFailed || t.Status == TurnInterrupted
}

type TurnRequest struct {
	ThreadID  string
	Input     string
	OnStarted func(Turn) error
}

type UnixClient struct {
	endpoint string
	nextID   atomic.Uint64
}

func NewUnixClient(endpoint string) *UnixClient {
	return &UnixClient{endpoint: endpoint}
}

func (c *UnixClient) StartThread(ctx context.Context, workspaceRoot string) (string, error) {
	connection, err := c.open(ctx)
	if err != nil {
		return "", err
	}
	defer connection.Close()

	var result struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := c.call(connection, "thread/start", threadStartParams{CWD: workspaceRoot}, &result); err != nil {
		return "", err
	}
	if result.Thread.ID == "" {
		return "", errors.New("app_response_missing_thread_id")
	}
	return result.Thread.ID, nil
}

func (c *UnixClient) open(ctx context.Context) (*Connection, error) {
	path, err := config.ControlSocketPath(c.endpoint)
	if err != nil {
		return nil, err
	}
	socket, err := new(net.Dialer).DialContext(ctx, "unix", path)
	if err != nil {
		return nil, fmt.Errorf("app_unreachable: %w", err)
	}
	connection := NewConnection(socket)
	if err := connection.Upgrade(ctx); err != nil {
		_ = connection.Close()
		return nil, err
	}
	if err := c.initialize(connection); err != nil {
		_ = connection.Close()
		return nil, err
	}
	return connection, nil
}

func (c *UnixClient) initialize(connection *Connection) error {
	params := initializeParams{
		ClientInfo: clientInfo{
			Name:    "codex-app-server-driver",
			Version: "v1",
		},
	}
	if err := c.notifyReady(connection, params); err != nil {
		return err
	}
	return connection.WriteJSON(rpcNotification{Version: "2.0", Method: "initialized"})
}

func (c *UnixClient) notifyReady(connection *Connection, params initializeParams) error {
	request := rpcRequest{
		Version: "2.0",
		ID:      c.nextID.Add(1),
		Method:  "initialize",
		Params:  params,
	}
	if err := connection.WriteJSON(request); err != nil {
		return err
	}
	var response rpcResponse
	if err := connection.ReadJSON(&response); err != nil {
		return err
	}
	if response.Error != nil {
		return fmt.Errorf("app_error: %s", response.Error.Message)
	}
	return nil
}

func (c *UnixClient) call(connection *Connection, method string, params any, result any) error {
	request := rpcRequest{
		Version: "2.0",
		ID:      c.nextID.Add(1),
		Method:  method,
		Params:  params,
	}
	if err := connection.WriteJSON(request); err != nil {
		return err
	}
	return decodeResponse(connection, result)
}

func (c *UnixClient) StartTurn(ctx context.Context, threadID string, input string) (Turn, error) {
	return c.RunTurn(ctx, TurnRequest{ThreadID: threadID, Input: input})
}

func (c *UnixClient) RunTurn(ctx context.Context, request TurnRequest) (Turn, error) {
	connection, err := c.open(ctx)
	if err != nil {
		return Turn{}, err
	}
	defer connection.Close()
	defer closeWithContext(ctx, connection)()

	message := rpcRequest{
		Version: "2.0",
		ID:      c.nextID.Add(1),
		Method:  "turn/start",
		Params: turnStartParams{
			ThreadID: request.ThreadID,
			Input:    textInputs(request.Input),
		},
	}
	if err := connection.WriteJSON(message); err != nil {
		return Turn{}, err
	}
	turn, err := decodeStartedTurn(connection)
	if err != nil {
		return Turn{}, err
	}
	if turn.ID == "" {
		return Turn{}, errors.New("app_response_missing_turn_id")
	}
	if request.OnStarted != nil {
		if err := request.OnStarted(turn); err != nil {
			return Turn{}, err
		}
	}
	if turn.Terminal() {
		return turn, nil
	}
	return waitForCompletion(connection, turn.ID)
}

func (c *UnixClient) InterruptTurn(ctx context.Context, turnID string) error {
	connection, err := c.open(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()

	var result map[string]any
	return c.call(connection, "turn/interrupt", map[string]any{"turnId": turnID}, &result)
}

func (c *UnixClient) Ready(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	connection, err := c.open(ctx)
	if err != nil {
		return false
	}
	_ = connection.Close()
	return true
}

func textInputs(value string) []userInput {
	return []userInput{{Type: "text", Text: value}}
}
