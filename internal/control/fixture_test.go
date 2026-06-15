package control

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type socketServer struct {
	listener           net.Listener
	methods            chan string
	done               chan struct{}
	syntheticThread    bool
	syntheticTurn      bool
	notifyBeforeThread bool
}

func newSocketServer(t *testing.T, socket string) *socketServer {
	t.Helper()

	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &socketServer{
		listener: listener,
		methods:  make(chan string, 8),
		done:     make(chan struct{}),
	}
	go server.accept()
	return server
}

func (s *socketServer) accept() {
	defer close(s.done)
	for {
		connection, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.respond(connection)
	}
}

func (s *socketServer) respond(connection net.Conn) {
	defer connection.Close()

	reader := bufio.NewReader(connection)
	if err := acceptUpgrade(reader, connection); err != nil {
		return
	}
	if !s.handleRequest(reader, connection, "initialize") {
		return
	}
	if !s.handleNotification(reader, "initialized") {
		return
	}
	s.handleNext(reader, connection)
}

func (s *socketServer) handleNext(reader *bufio.Reader, connection net.Conn) {
	var request map[string]any
	if err := readTextJSON(reader, &request); err != nil {
		return
	}
	method, _ := request["method"].(string)
	s.methods <- method
	if err := validateParams(method, request["params"]); err != nil {
		writeError(connection, request["id"], err)
		return
	}
	if s.notifyBeforeThread && method == "thread/start" {
		writeTextJSON(connection, map[string]any{
			"jsonrpc": "2.0",
			"method":  "thread/started",
			"params":  threadStartResult(),
		})
	}
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      request["id"],
		"result":  resultFor(method),
	}
	if s.syntheticThread && method == "thread/start" {
		response["result"] = map[string]any{"thread_id": "3c9e5a1b7d04f268"}
	}
	if s.syntheticTurn && method == "turn/start" {
		response["result"] = map[string]any{"turn_id": "4dab6c2e8f105379", "completed": true}
	}
	writeTextJSON(connection, response)
}

func (s *socketServer) handleRequest(reader *bufio.Reader, connection net.Conn, want string) bool {
	var request map[string]any
	if err := readTextJSON(reader, &request); err != nil {
		return false
	}
	method, _ := request["method"].(string)
	s.methods <- method
	if method != want {
		writeError(connection, request["id"], fmt.Errorf("method must be %s", want))
		return false
	}
	if err := validateParams(method, request["params"]); err != nil {
		writeError(connection, request["id"], err)
		return false
	}
	writeTextJSON(connection, map[string]any{
		"jsonrpc": "2.0",
		"id":      request["id"],
		"result":  map[string]any{},
	})
	return true
}

func (s *socketServer) handleNotification(reader *bufio.Reader, want string) bool {
	var request map[string]any
	if err := readTextJSON(reader, &request); err != nil {
		return false
	}
	method, _ := request["method"].(string)
	s.methods <- method
	return method == want
}

func acceptUpgrade(reader *bufio.Reader, connection net.Conn) error {
	request, err := http.ReadRequest(reader)
	if err != nil {
		return err
	}
	key := request.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return errors.New("missing websocket key")
	}
	_, err = fmt.Fprintf(
		connection,
		"HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",
		serverAcceptKey(key),
	)
	return err
}

func serverAcceptKey(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func readTextJSON(reader *bufio.Reader, value any) error {
	payload, err := readFrame(reader)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, value)
}

func writeTextJSON(writer io.Writer, value any) {
	content, _ := json.Marshal(value)
	_ = writeFrame(writer, content)
}

func readFrame(reader *bufio.Reader) ([]byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}
	length := int(header[1] & 0x7f)
	if length == 126 {
		extended := make([]byte, 2)
		if _, err := io.ReadFull(reader, extended); err != nil {
			return nil, err
		}
		length = int(binary.BigEndian.Uint16(extended))
	}
	mask := make([]byte, 4)
	if header[1]&0x80 != 0 {
		if _, err := io.ReadFull(reader, mask); err != nil {
			return nil, err
		}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	for index := range payload {
		payload[index] ^= mask[index%4]
	}
	return payload, nil
}

func writeFrame(writer io.Writer, payload []byte) error {
	frame := appendLength([]byte{0x81}, len(payload), false)
	frame = append(frame, payload...)
	_, err := writer.Write(frame)
	return err
}

func validateParams(method string, params any) error {
	values, ok := params.(map[string]any)
	if !ok {
		return errors.New("params must be an object")
	}
	if method == "initialize" {
		return validateInitializeParams(values)
	}
	if method == "thread/start" {
		return validateThreadParams(values)
	}
	if method == "turn/start" {
		return validateTurnParams(values)
	}
	return nil
}

func validateInitializeParams(params map[string]any) error {
	client, ok := params["clientInfo"].(map[string]any)
	if !ok {
		return errors.New("clientInfo is required")
	}
	if client["name"] != "codex-app-server-driver" {
		return errors.New("clientInfo.name must identify the selected driver")
	}
	if client["version"] != "v1" {
		return errors.New("clientInfo.version must identify the selected driver contract")
	}
	return nil
}

func validateThreadParams(params map[string]any) error {
	if _, ok := params["workspace_root"]; ok {
		return errors.New("workspace_root is not a thread/start param")
	}
	cwd, _ := params["cwd"].(string)
	if cwd != "/tmp/work" {
		return errors.New("cwd must carry the workspace root")
	}
	return nil
}

func validateTurnParams(params map[string]any) error {
	if _, ok := params["thread_id"]; ok {
		return errors.New("thread_id is not a turn/start param")
	}
	if threadID, _ := params["threadId"].(string); threadID != "3c9e5a1b7d04f268" {
		return errors.New("threadId must carry the thread identifier")
	}
	inputs, ok := params["input"].([]any)
	if !ok {
		return errors.New("input must be a UserInput array")
	}
	return validateTextInput(inputs)
}

func validateTextInput(inputs []any) error {
	if len(inputs) != 1 {
		return errors.New("exactly one text input is required")
	}
	input, ok := inputs[0].(map[string]any)
	if !ok {
		return errors.New("input item must be an object")
	}
	if input["type"] != "text" || input["text"] != "run close" {
		return errors.New("input item must be text UserInput")
	}
	return nil
}

func writeError(connection net.Conn, id any, err error) {
	writeTextJSON(connection, map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    -32602,
			"message": err.Error(),
		},
	})
}

func resultFor(method string) map[string]any {
	if method == "thread/start" {
		return threadStartResult()
	}
	return turnResult("completed")
}

func threadStartResult() map[string]any {
	return map[string]any{
		"approvalPolicy":    "never",
		"approvalsReviewer": "user",
		"cwd":               "/tmp/work",
		"model":             "7a0d2c4e6f8193b5",
		"modelProvider":     "8b1e3d5f7092a4c6",
		"sandbox":           map[string]any{"mode": "read-only"},
		"thread": map[string]any{
			"cliVersion":    "0.0.0",
			"createdAt":     1,
			"cwd":           "/tmp/work",
			"ephemeral":     true,
			"id":            "3c9e5a1b7d04f268",
			"modelProvider": "8b1e3d5f7092a4c6",
			"preview":       "",
			"sessionId":     "5ebc7d3f9012648a",
			"source":        "app-server",
			"status":        "idle",
			"turns":         []any{},
			"updatedAt":     1,
		},
	}
}

func turnResult(status string) map[string]any {
	return map[string]any{"turn": map[string]any{
		"id":     "4dab6c2e8f105379",
		"items":  []any{},
		"status": status,
	}}
}

func (s *socketServer) Methods() []string {
	var methods []string
	for len(s.methods) > 0 {
		methods = append(methods, <-s.methods)
	}
	return methods
}

func (s *socketServer) Close() {
	_ = s.listener.Close()
	<-s.done
}

func writeFile(t *testing.T, root string, path string, content string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func requireMethods(t *testing.T, actual []string, expected []string) {
	t.Helper()

	if !slices.Equal(actual, expected) {
		t.Fatalf("expected methods %v, got %v", expected, actual)
	}
}

func requireError(t *testing.T, err error, expected string) {
	t.Helper()

	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error containing %q, got %v", expected, err)
	}
}

func requireCommand(t *testing.T, actual Command, expected Command) {
	t.Helper()

	if actual.Name != expected.Name || !slices.Equal(actual.Args, expected.Args) {
		t.Fatalf("command = %+v, want name=%s args=%v", actual, expected.Name, expected.Args)
	}
	for _, value := range expected.Env {
		if !slices.Contains(actual.Env, value) {
			t.Fatalf("command = %+v, want env containing %s", actual, value)
		}
	}
	if slices.Contains(actual.Env, "CODEX_HOME=/tmp/ambient-codex-home") {
		t.Fatalf("command used ambient CODEX_HOME: %+v", actual)
	}
}
