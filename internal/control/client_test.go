package control

import (
	"context"
	"path/filepath"
	"testing"
)

func TestUnixClientSendsThreadAndTurnMethods(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "control.sock")
	server := newSocketServer(t, socket)
	defer server.Close()

	client := NewUnixClient("unix://" + socket)
	threadID, err := client.StartThread(context.Background(), "/tmp/work")
	if err != nil {
		t.Fatal(err)
	}
	turn, err := client.StartTurn(context.Background(), threadID, "run close")
	if err != nil {
		t.Fatal(err)
	}

	if threadID != "3c9e5a1b7d04f268" || turn.ID != "4dab6c2e8f105379" || !turn.Completed() {
		t.Fatalf("unexpected thread or turn result: thread=%s turn=%+v", threadID, turn)
	}
	requireMethods(t, server.Methods(), []string{
		"initialize",
		"initialized",
		"thread/start",
		"initialize",
		"initialized",
		"turn/start",
	})
}

func TestUnixClientRejectsSyntheticTopLevelThreadResponse(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "control.sock")
	server := newSocketServer(t, socket)
	server.syntheticThread = true
	defer server.Close()

	client := NewUnixClient("unix://" + socket)
	_, err := client.StartThread(context.Background(), "/tmp/work")

	requireError(t, err, "app_response_missing_thread_id")
}

func TestUnixClientRejectsSyntheticTopLevelTurnResponse(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "control.sock")
	server := newSocketServer(t, socket)
	server.syntheticTurn = true
	defer server.Close()

	client := NewUnixClient("unix://" + socket)
	_, err := client.StartTurn(context.Background(), "3c9e5a1b7d04f268", "run close")

	requireError(t, err, "app_response_missing_turn_id")
}

func TestUnixClientIgnoresThreadNotificationBeforeResponse(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "control.sock")
	server := newSocketServer(t, socket)
	server.notifyBeforeThread = true
	defer server.Close()

	client := NewUnixClient("unix://" + socket)
	threadID, err := client.StartThread(context.Background(), "/tmp/work")

	if err != nil {
		t.Fatal(err)
	}
	if threadID != "3c9e5a1b7d04f268" {
		t.Fatalf("thread id = %q", threadID)
	}
}
