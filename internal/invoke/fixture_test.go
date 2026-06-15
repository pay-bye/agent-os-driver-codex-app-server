package invoke

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

type httpServer struct {
	*httptest.Server
	routes                 []string
	bodies                 map[string]map[string]any
	empty                  bool
	malformedMetadata      bool
	omitCompatibilityRoute bool
	unknownMetadataRoute   bool
	nack                   map[string]any
}

func newHTTPServer(t *testing.T) *httpServer {
	t.Helper()

	server := &httpServer{bodies: map[string]map[string]any{}}
	server.Server = httptest.NewServer(http.HandlerFunc(server.handle))
	return server
}

func (s *httpServer) handle(writer http.ResponseWriter, request *http.Request) {
	s.routes = append(s.routes, request.URL.Path)
	switch request.URL.Path {
	case "/claim":
		s.writeClaim(writer)
	case "/compatibility":
		s.writeMetadata(writer)
	case "/extend", "/ack":
		s.recordBody(request)
		writeJSON(writer, map[string]any{"resolved": true, "routed": false})
	case "/nack":
		s.recordBody(request)
		s.nack = s.bodies["/nack"]
		writeJSON(writer, map[string]any{"resolved": true, "routed": false})
	default:
		http.NotFound(writer, request)
	}
}

func (s *httpServer) writeMetadata(writer http.ResponseWriter) {
	if s.malformedMetadata {
		writeJSON(writer, map[string]any{"contract_version": "x91"})
		return
	}
	routes := []map[string]string{
		{"method": "POST", "path": "/claim"},
		{"method": "POST", "path": "/ack"},
		{"method": "POST", "path": "/nack"},
		{"method": "POST", "path": "/extend"},
		{"method": "GET", "path": "/compatibility"},
	}
	if s.omitCompatibilityRoute {
		routes = routes[:len(routes)-1]
	}
	if s.unknownMetadataRoute {
		routes = append(routes, map[string]string{"method": "POST", "path": "/unknown"})
	}
	writeJSON(writer, map[string]any{
		"contract_version":  "v1",
		"schema_set_digest": "sha256:19839901e6b07f949d015821f5f6823fc2fc98fdce4934904294f6c75404f9ac",
		"features": []string{
			"lease_claim",
			"lease_extend",
			"lease_ack",
			"lease_nack",
			"lease_capability",
			"declared_needs",
			"failure_payload",
		},
		"routes": routes,
	})
}

func (s *httpServer) writeClaim(writer http.ResponseWriter) {
	if s.empty {
		writeJSON(writer, map[string]any{"empty": true})
		return
	}
	writeJSON(writer, map[string]any{
		"empty":        false,
		"lease_id":     "1a7c3e9b5f024d68",
		"lease_token":  "x-token",
		"work_item_id": "2b8d4f0a6c93e157",
		"payload":      map[string]any{"work": map[string]any{"prompt": "run close"}},
		"expires_at":   time.Unix(90, 0).UTC().Format(time.RFC3339),
	})
}

func (s *httpServer) Routes() []string {
	return slices.Clone(s.routes)
}

func (s *httpServer) Token(path string) string {
	body := s.bodies[path]
	value, _ := body["lease_token"].(string)
	return value
}

func (s *httpServer) recordBody(request *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(request.Body).Decode(&body)
	s.bodies[request.URL.Path] = body
}

func (s *httpServer) NackNeed() string {
	items, _ := s.nack["declared_needs"].([]any)
	if len(items) == 0 {
		return ""
	}
	item, _ := items[0].(map[string]any)
	kind, _ := item["need_kind"].(string)
	return kind
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
