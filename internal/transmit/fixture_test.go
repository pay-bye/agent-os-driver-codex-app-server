package transmit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/control"
)

type invocationServer struct {
	*httptest.Server
	claim         map[string]any
	claimBodies   []map[string]any
	ack           map[string]any
	failExtend    bool
	routes        []string
	nack          map[string]any
	extensions    []map[string]any
	extended      chan struct{}
	extendGoal    int
	extensionEnds []time.Time
}

func newInvocationServer(t *testing.T, claim map[string]any) *invocationServer {
	t.Helper()

	server := &invocationServer{claim: claim, extended: make(chan struct{})}
	server.extendGoal = 1
	if !claimEmpty(claim) {
		server.extensionEnds = []time.Time{claimExpiresAt(t, claim)}
	}
	server.Server = httptest.NewServer(http.HandlerFunc(server.handle))
	return server
}

func (s *invocationServer) handle(writer http.ResponseWriter, request *http.Request) {
	s.routes = append(s.routes, request.URL.Path)
	switch request.URL.Path {
	case "/claim":
		s.recordClaim(request)
		writeJSON(writer, s.claim)
	case "/extend":
		if !s.recordExtension(request) {
			http.Error(writer, `{"error":"non_increasing_extension"}`, http.StatusConflict)
			return
		}
		if s.failExtend {
			http.Error(writer, `{"error":"expired_lease"}`, http.StatusNotFound)
			return
		}
		writeJSON(writer, map[string]any{"extended": true})
	case "/ack":
		_ = json.NewDecoder(request.Body).Decode(&s.ack)
		writeJSON(writer, map[string]any{"resolved": true, "routed": false})
	case "/nack":
		_ = json.NewDecoder(request.Body).Decode(&s.nack)
		writeJSON(writer, map[string]any{"resolved": true, "routed": false})
	default:
		http.NotFound(writer, request)
	}
}

func (s *invocationServer) recordClaim(request *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(request.Body).Decode(&body)
	s.claimBodies = append(s.claimBodies, body)
}

func (s *invocationServer) ClaimID() string {
	if len(s.claimBodies) == 0 {
		return ""
	}
	value, _ := s.claimBodies[0]["lease_id"].(string)
	return value
}

func (s *invocationServer) Extended() <-chan struct{} {
	return s.extended
}

func (s *invocationServer) markExtended() {
	if len(s.extensionEnds) < s.extendGoal+1 {
		return
	}
	select {
	case <-s.extended:
	default:
		close(s.extended)
	}
}

func (s *invocationServer) recordExtension(request *http.Request) bool {
	var body map[string]any
	_ = json.NewDecoder(request.Body).Decode(&body)
	s.extensions = append(s.extensions, body)
	requested, ok := extensionTime(body)
	if !ok || !requested.After(s.currentExpiry()) {
		return false
	}
	s.extensionEnds = append(s.extensionEnds, requested)
	s.markExtended()
	return true
}

func (s *invocationServer) currentExpiry() time.Time {
	return s.extensionEnds[len(s.extensionEnds)-1]
}

func (s *invocationServer) Routes() []string {
	return slices.Clone(s.routes)
}

func (s *invocationServer) NackNeed() string {
	items, _ := s.nack["declared_needs"].([]any)
	if len(items) == 0 {
		return ""
	}
	item, _ := items[0].(map[string]any)
	kind, _ := item["need_kind"].(string)
	return kind
}

func (s *invocationServer) AckToken() string {
	value, _ := s.ack["lease_token"].(string)
	return value
}

func (s *invocationServer) NackToken() string {
	value, _ := s.nack["lease_token"].(string)
	return value
}

func (s *invocationServer) ExtensionToken() string {
	if len(s.extensions) == 0 {
		return ""
	}
	value, _ := s.extensions[0]["lease_token"].(string)
	return value
}

type appServer struct {
	mu                sync.Mutex
	methods           []string
	completionGate    <-chan struct{}
	cancelAfterStart  context.CancelFunc
	delayedCompletion bool
	failTurn          bool
}

func newAppServer(t *testing.T) *appServer {
	t.Helper()

	return &appServer{}
}

func (s *appServer) Endpoint() string {
	return "unix:///tmp/test-owned-control.sock"
}

func (s *appServer) StartThread(_ context.Context, workspaceRoot string) (string, error) {
	s.recordMethod("thread/start")
	if err := validateThreadParams(map[string]any{"cwd": workspaceRoot}); err != nil {
		return "", err
	}
	return "3c9e5a1b7d04f268", nil
}

func (s *appServer) RunTurn(_ context.Context, request control.TurnRequest) (control.Turn, error) {
	s.recordMethod("turn/start")
	if err := validateTurnRequest(request); err != nil {
		return control.Turn{}, err
	}
	if request.OnStarted != nil {
		if err := request.OnStarted(control.Turn{ID: "4dab6c2e8f105379", Status: control.TurnInProgress}); err != nil {
			return control.Turn{}, err
		}
	}
	if s.cancelAfterStart != nil {
		s.cancelAfterStart()
	}
	status := control.TurnCompleted
	if s.failTurn {
		status = control.TurnFailed
	}
	if s.delayedCompletion {
		s.waitForExtension()
	}
	return control.Turn{ID: "4dab6c2e8f105379", Status: status}, nil
}

func (s *appServer) InterruptTurn(_ context.Context, turnID string) error {
	s.recordMethod("turn/interrupt")
	if turnID == "" {
		return errors.New("turn id required")
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

func validateTurnRequest(request control.TurnRequest) error {
	if request.ThreadID != "3c9e5a1b7d04f268" {
		return errors.New("threadId must carry the thread identifier")
	}
	if request.Input != "run close" {
		return errors.New("input item must be text UserInput")
	}
	return nil
}

func (s *appServer) waitForExtension() {
	if s.completionGate == nil {
		return
	}
	select {
	case <-s.completionGate:
	case <-time.After(200 * time.Millisecond):
	}
}

func (s *appServer) Methods() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.methods)
}

func (s *appServer) Close() {}

func (s *appServer) recordMethod(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.methods = append(s.methods, method)
}

type fixedCheck struct {
	result compatibility.Result
	err    error
}

type failingCheck fixedCheck

func successfulCheck() fixedCheck {
	return fixedCheck{result: compatibility.Result{DiagnosticCode: "compatible"}}
}

func (c fixedCheck) Verify(context.Context) (compatibility.Result, error) {
	return c.result, c.err
}

func (c failingCheck) Verify(context.Context) (compatibility.Result, error) {
	return compatibility.Result{DiagnosticCode: compatibility.Code(c.err)}, c.err
}

func claimWithPayload(payload map[string]any) map[string]any {
	return map[string]any{
		"empty":        false,
		"lease_id":     "1a7c3e9b5f024d68",
		"lease_token":  "x-token",
		"work_item_id": "2b8d4f0a6c93e157",
		"payload":      payload,
		"expires_at":   time.Unix(90, 0).UTC().Format(time.RFC3339),
	}
}

func claimExpiresAt(t *testing.T, claim map[string]any) time.Time {
	t.Helper()

	value, _ := claim["expires_at"].(string)
	expiresAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return expiresAt
}

func claimEmpty(claim map[string]any) bool {
	empty, _ := claim["empty"].(bool)
	return empty
}

func extensionTime(body map[string]any) (time.Time, bool) {
	value, ok := body["requested_expires_at"].(string)
	if !ok {
		return time.Time{}, false
	}
	expiresAt, err := time.Parse(time.RFC3339, value)
	return expiresAt, err == nil
}

func validConfig(baseURL string, endpoint string) config.Config {
	return config.Config{
		InvocationBaseURL: baseURL,
		ChannelKey:        "q01",
		LeaseSeconds:      60,
		CodexHome:         "/tmp/codex-home",
		ControlEndpoint:   endpoint,
		WorkspaceRoot:     "/tmp/work",
		InputTextPointer:  "/work/prompt",
		CompletionNeeds:   []config.Need{{Kind: "done"}},
		FailureNeeds:      []config.Need{{Kind: "failed"}},
		RedactionMode:     "metadata_only",
	}
}

func writeJSON(writer http.ResponseWriter, value any) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(value)
}

func requireRoutes(t *testing.T, actual []string, expected []string) {
	t.Helper()

	if !slices.Equal(actual, expected) {
		t.Fatalf("expected routes %v, got %v", expected, actual)
	}
}

func requireMethods(t *testing.T, actual []string, expected []string) {
	t.Helper()

	if !slices.Equal(actual, expected) {
		t.Fatalf("expected methods %v, got %v", expected, actual)
	}
}

func requireHexLength(t *testing.T, value string, length int) {
	t.Helper()

	pattern := regexp.MustCompile(`^[0-9a-f]+$`)
	if len(value) != length || !pattern.MatchString(value) {
		t.Fatalf("id = %q, want %d lowercase hex chars", value, length)
	}
}
