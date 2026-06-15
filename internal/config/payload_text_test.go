package config

import "testing"

func TestExtractPromptTextResolvesPointer(t *testing.T) {
	payload := map[string]any{"work": map[string]any{"prompt": "run close"}}

	text, err := ExtractPromptText(payload, "/work/prompt")

	if err != nil {
		t.Fatal(err)
	}
	if text != "run close" {
		t.Fatalf("expected prompt text, got %q", text)
	}
}

func TestExtractPromptTextRejectsMissingString(t *testing.T) {
	payload := map[string]any{"work": map[string]any{"prompt": 42}}

	_, err := ExtractPromptText(payload, "/work/prompt")

	requireError(t, err, "invalid_payload")
}
