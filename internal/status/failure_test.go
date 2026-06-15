package status

import "testing"

func TestFailurePayloadContainsOnlyErrorCode(t *testing.T) {
	payload := FailurePayload("invalid_payload", map[string]any{
		"prompt":     "prompt-body",
		"credential": "private-value",
	})

	if payload["error_code"] != "invalid_payload" {
		t.Fatalf("unexpected payload %+v", payload)
	}
	if len(payload) != 1 {
		t.Fatalf("expected one redacted field, got %+v", payload)
	}
	if containsValue(payload, "prompt-body") || containsValue(payload, "private-value") {
		t.Fatalf("failure payload exposed source values: %+v", payload)
	}
}

func containsValue(payload map[string]any, expected string) bool {
	for _, value := range payload {
		if value == expected {
			return true
		}
	}
	return false
}
