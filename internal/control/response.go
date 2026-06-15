package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type rpcResponse struct {
	Method string          `json:"method,omitempty"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type turnStartResult struct {
	Turn Turn `json:"turn"`
}

type notification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type completedParams struct {
	Turn Turn `json:"turn"`
}

func decodeResponse(connection *Connection, result any) error {
	for {
		var response rpcResponse
		if err := connection.ReadJSON(&response); err != nil {
			return err
		}
		if response.Method != "" {
			continue
		}
		if response.Error != nil {
			return fmt.Errorf("app_error: %s", response.Error.Message)
		}
		if len(response.Result) == 0 {
			return errors.New("app_response_missing_result")
		}
		return json.Unmarshal(response.Result, result)
	}
}

func decodeStartedTurn(connection *Connection) (Turn, error) {
	var response rpcResponse
	if err := connection.ReadJSON(&response); err != nil {
		return Turn{}, err
	}
	if response.Error != nil {
		return Turn{}, fmt.Errorf("app_error: %s", response.Error.Message)
	}
	var result turnStartResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return Turn{}, err
	}
	return result.Turn, nil
}

func waitForCompletion(connection *Connection, turnID string) (Turn, error) {
	for {
		var item notification
		if err := connection.ReadJSON(&item); err != nil {
			return Turn{}, err
		}
		if item.Method != "turn/completed" {
			continue
		}
		turn, err := completedTurn(item.Params)
		if err != nil {
			return Turn{}, err
		}
		if turn.ID == turnID {
			return turn, nil
		}
	}
}

func closeWithContext(ctx context.Context, connection *Connection) func() {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = connection.Close()
		case <-done:
		}
	}()
	return func() {
		close(done)
	}
}

func completedTurn(content json.RawMessage) (Turn, error) {
	var params completedParams
	if err := json.Unmarshal(content, &params); err != nil {
		return Turn{}, err
	}
	if params.Turn.ID == "" {
		return Turn{}, errors.New("app_response_missing_turn_id")
	}
	return params.Turn, nil
}
