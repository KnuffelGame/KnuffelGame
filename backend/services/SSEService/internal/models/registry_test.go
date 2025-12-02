package models

import (
	"encoding/json"
	"testing"
	"time"
)

// helper: receive SSEMessage from ch within d or fail
func recvMsg(t *testing.T, ch <-chan SSEMessage, d time.Duration) SSEMessage {
	t.Helper()
	select {
	case m := <-ch:
		return m
	case <-time.After(d):
		t.Fatalf("timeout waiting for message")
		return SSEMessage{}
	}
}

// helper: expect Done signal within d
func expectDone(t *testing.T, done <-chan struct{}, d time.Duration) {
	t.Helper()
	select {
	case <-done:
		return
	case <-time.After(d):
		t.Fatalf("timeout waiting for Done signal")
	}
}

// helper: expect no message within d
func expectNoRecv(t *testing.T, ch <-chan SSEMessage, d time.Duration) {
	t.Helper()
	select {
	case m := <-ch:
		t.Fatalf("unexpected message: %+v", m)
	case <-time.After(d):
		return
	}
}

func TestEnsureTarget_Idempotent(t *testing.T) {
	r := NewRegistry()
	te1 := r.EnsureTarget(TargetTypeLobby, "lby_1")
	if te1 == nil {
		t.Fatal("EnsureTarget returned nil")
	}
	te2 := r.EnsureTarget(TargetTypeLobby, "lby_1")
	if te2 == nil {
		t.Fatal("EnsureTarget returned nil on second call")
	}
	if te1 != te2 {
		t.Fatalf("EnsureTarget not idempotent: pointers differ")
	}
	if te1.Type != TargetTypeLobby || te1.ID != "lby_1" {
		t.Fatalf("unexpected target fields: %+v", te1)
	}
}

func TestRegisterConnection_AddsAndDuplicateReplace(t *testing.T) {
	r := NewRegistry()
	te, c1 := r.RegisterConnection(TargetTypeLobby, "lby_a", "usr1")
	if te == nil || c1 == nil {
		t.Fatal("RegisterConnection returned nil")
	}
	if te.Connections["usr1"] != c1 {
		t.Fatalf("connection not stored in map")
	}
	// duplicate registration for same userID should replace stored connection
	te2, c2 := r.RegisterConnection(TargetTypeLobby, "lby_a", "usr1")
	if te2 != te {
		t.Fatalf("target entry mismatch on duplicate register")
	}
	if c2 == c1 {
		t.Fatalf("duplicate user registration did not replace the connection")
	}
	if te.Connections["usr1"] != c2 {
		t.Fatalf("map does not contain replaced connection")
	}
	// original connection should not receive any implicit Done signal here
	expectNoRecv(t, c1.Ch, 10*time.Millisecond)
}

func TestBroadcast_AllAndTargeted_UpdatesMetricsAndLastEvent(t *testing.T) {
	r := NewRegistry()
	te, c1 := r.RegisterConnection(TargetTypeLobby, "lby_b", "u1")
	_, c2 := r.RegisterConnection(TargetTypeLobby, "lby_b", "u2")

	// broadcast to all
	payload := map[string]any{"x": 1}
	data, _ := json.Marshal(payload)
	found, sent, failed, ok := r.Broadcast(TargetTypeLobby, "lby_b", SSEMessage{Event: "e1", Data: data}, "")
	if !ok {
		t.Fatalf("broadcast reported target missing")
	}
	if found != 2 || sent != 2 || failed != 0 {
		t.Fatalf("broadcast all counters mismatch: found=%d sent=%d failed=%d", found, sent, failed)
	}

	// both connections should receive the message
	m1 := recvMsg(t, c1.Ch, 50*time.Millisecond)
	m2 := recvMsg(t, c2.Ch, 50*time.Millisecond)
	if m1.Event != "e1" || string(m1.Data) != string(data) || m2.Event != "e1" || string(m2.Data) != string(data) {
		t.Fatalf("messages mismatch: m1=%+v m2=%+v", m1, m2)
	}

	// targeted broadcast to u2 only
	payload2 := map[string]any{"y": "ok"}
	data2, _ := json.Marshal(payload2)
	found, sent, failed, ok = r.Broadcast(TargetTypeLobby, "lby_b", SSEMessage{Event: "e2", Data: data2}, "u2")
	if !ok || found != 1 || sent != 1 || failed != 0 {
		t.Fatalf("targeted broadcast counters mismatch: found=%d sent=%d failed=%d ok=%v", found, sent, failed, ok)
	}
	// u2 receives, u1 does not (within small window)
	m2t := recvMsg(t, c2.Ch, 50*time.Millisecond)
	if m2t.Event != "e2" || string(m2t.Data) != string(data2) {
		t.Fatalf("u2 targeted message mismatch: %+v", m2t)
	}
	expectNoRecv(t, c1.Ch, 20*time.Millisecond)

	// metrics and LastEventAt updated
	if te.LastEventAt.IsZero() {
		t.Fatalf("LastEventAt not updated")
	}
	if te.Metrics.EventsSent < 3 {
		t.Fatalf("EventsSent not incremented as expected, got %d", te.Metrics.EventsSent)
	}
	if te.Metrics.FailedConnections != 0 {
		t.Fatalf("FailedConnections expected 0, got %d", te.Metrics.FailedConnections)
	}
}

func TestBroadcast_NonBlockingSend_FullChannel_SchedulesCleanup(t *testing.T) {
	r := NewRegistry()
	te, c := r.RegisterConnection(TargetTypeLobby, "lby_full", "u1")

	// Fill the buffered channel to its capacity (16)
	fill := SSEMessage{Event: "fill", Data: []byte(`{}`)}
	for i := 0; i < 16; i++ {
		select {
		case c.Ch <- fill:
		case <-time.After(50 * time.Millisecond):
			t.Fatalf("failed to pre-fill channel")
		}
	}

	// Now broadcast should fail to send and schedule removal
	found, sent, failed, ok := r.Broadcast(TargetTypeLobby, "lby_full", SSEMessage{Event: "e", Data: []byte(`{}`)}, "")
	if !ok {
		t.Fatalf("broadcast reported missing target")
	}
	if found != 1 || sent != 0 || failed != 1 {
		t.Fatalf("expected found=1 sent=0 failed=1, got %d,%d,%d", found, sent, failed)
	}

	// Expect Done signal due to scheduled RemoveConnection
	expectDone(t, c.Done, 200*time.Millisecond)

	// Metrics updated
	if te.Metrics.FailedConnections < 1 {
		t.Fatalf("FailedConnections not incremented, got %d", te.Metrics.FailedConnections)
	}
}

func TestUnregisterTarget_ClosesAllAndRemoves(t *testing.T) {
	r := NewRegistry()
	_, c1 := r.RegisterConnection(TargetTypeGame, "gam_x", "a")
	_, c2 := r.RegisterConnection(TargetTypeGame, "gam_x", "b")

	closed, ok := r.UnregisterTarget(TargetTypeGame, "gam_x")
	if !ok {
		t.Fatalf("UnregisterTarget returned ok=false")
	}
	if closed != 2 {
		t.Fatalf("connections_closed=%d, want 2", closed)
	}

	// both Done channels should have been signaled
	expectDone(t, c1.Done, 50*time.Millisecond)
	expectDone(t, c2.Done, 50*time.Millisecond)

	// target should be removed
	if r.HasTarget(TargetTypeGame, "gam_x") {
		t.Fatalf("target still present after UnregisterTarget")
	}
}

func TestStats_Aggregation(t *testing.T) {
	r := NewRegistry()
	// lobby with 2
	r.RegisterConnection(TargetTypeLobby, "lby_s1", "u1")
	r.RegisterConnection(TargetTypeLobby, "lby_s1", "u2")
	// game with 1
	r.RegisterConnection(TargetTypeGame, "gam_s1", "u3")

	totalTargets, totalConns, lobbyTargets, lobbyConns, gameTargets, gameConns := r.Stats()
	if totalTargets != 2 || totalConns != 3 || lobbyTargets != 1 || lobbyConns != 2 || gameTargets != 1 || gameConns != 1 {
		t.Fatalf("stats mismatch: totals(%d,%d) lobby(%d,%d) game(%d,%d)", totalTargets, totalConns, lobbyTargets, lobbyConns, gameTargets, gameConns)
	}
}

func TestBroadcast_TargetMissing(t *testing.T) {
	r := NewRegistry()
	found, sent, failed, ok := r.Broadcast(TargetTypeLobby, "does_not_exist", SSEMessage{Event: "e", Data: []byte(`{}`)}, "")
	if ok {
		t.Fatalf("expected ok=false for missing target")
	}
	if found != 0 || sent != 0 || failed != 0 {
		t.Fatalf("unexpected counters for missing target: %d,%d,%d", found, sent, failed)
	}
}
