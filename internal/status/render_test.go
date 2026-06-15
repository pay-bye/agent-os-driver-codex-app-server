package status

import (
	"strings"
	"testing"
)

func TestRenderIncludesMetadataCountersOnly(t *testing.T) {
	output := Render(Counts{
		Source:        "local",
		InstallID:     "6fcd8a420b3e9571",
		ClaimAttempts: 2,
		EmptyClaims:   1,
		ActiveLeaseID: "1a7c3e9b5f024d68",
		WorkItemID:    "2b8d4f0a6c93e157",
		ThreadID:      "3c9e5a1b7d04f268",
		TurnID:        "4dab6c2e8f105379",
		Acks:          1,
		Nacks:         1,
		Extensions:    1,
		LastErrorCode: "invalid_payload",
	})

	requireContains(t, output, "6fcd8a420b3e9571")
	requireContains(t, output, "source=local")
	requireContains(t, output, "claim_attempts=2")
}

func TestRenderDiagnosticNamesSourceAndCode(t *testing.T) {
	output := RenderDiagnostic("live", "route_missing")

	requireContains(t, output, "source=live")
	requireContains(t, output, "error_code=route_missing")
}

func requireContains(t *testing.T, content string, expected string) {
	t.Helper()

	if !strings.Contains(content, expected) {
		t.Fatalf("expected %q in %q", expected, content)
	}
}
