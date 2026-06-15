package config

import "testing"

func TestReadAcceptsJSONConfigWithUnixEndpoint(t *testing.T) {
	path := writeConfig(t, validConfig())

	item, err := Read(path, acceptCodex)

	if err != nil {
		t.Fatal(err)
	}
	if item.ChannelKey != "q01" {
		t.Fatalf("expected channel q01, got %q", item.ChannelKey)
	}
}
