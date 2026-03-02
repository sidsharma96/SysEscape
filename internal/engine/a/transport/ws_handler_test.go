package transport

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sidsharma96/SysEscape/internal/token"
	"github.com/sidsharma96/SysEscape/internal/ws"
)

func TestWSHandlerHelloAuthAndSnapshot(t *testing.T) {
	runID := uuid.New()
	secret := "secret"
	tok := mustMintToken(t, secret, runID)

	runtime := &fakeRuntime{connectResult: ConnectResult{
		SnapshotRequired: true,
		Snapshot:         ws.Delta{Seq: 4, Payload: json.RawMessage(`{"metrics":{"cache_hit_rate":0.9}}`)},
	}}

	h := NewWSHandler(HandlerConfig{Secret: secret, Runtime: runtime})
	s := httptest.NewServer(h)
	defer s.Close()

	conn := mustDial(t, s.URL, runID)
	defer conn.Close()

	mustWrite(t, conn, ws.Envelope{ProtocolVersion: ws.ProtocolVersion1, Type: ws.TypeHello, RunID: runID.String(), Payload: mustRawMsg(t, ws.HelloPayload{RunToken: tok})})

	ack := mustRead(t, conn)
	if ack.Type != ws.TypeHelloAck {
		t.Fatalf("ack type=%q", ack.Type)
	}
	var ackPayload ws.HelloAckPayload
	mustDecode(t, ack.Payload, &ackPayload)
	if !ackPayload.SnapshotRequired {
		t.Fatalf("snapshotRequired=false")
	}

	snapshot := mustRead(t, conn)
	if snapshot.Type != ws.TypeSnapshot || snapshot.Seq == nil || *snapshot.Seq != 4 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestWSHandlerRejectsInvalidToken(t *testing.T) {
	runID := uuid.New()
	h := NewWSHandler(HandlerConfig{Secret: "secret", Runtime: &fakeRuntime{}})
	s := httptest.NewServer(h)
	defer s.Close()

	conn := mustDial(t, s.URL, runID)
	defer conn.Close()

	mustWrite(t, conn, ws.Envelope{ProtocolVersion: ws.ProtocolVersion1, Type: ws.TypeHello, RunID: runID.String(), Payload: mustRawMsg(t, ws.HelloPayload{RunToken: "bad"})})
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := conn.ReadMessage()
	if err == nil {
		t.Fatalf("expected close on invalid token")
	}
}

func TestWSHandlerApplyActionAndDelta(t *testing.T) {
	runID := uuid.New()
	secret := "secret"
	tok := mustMintToken(t, secret, runID)

	runtime := &fakeRuntime{
		connectResult: ConnectResult{SnapshotRequired: true, Snapshot: ws.Delta{Seq: 1, Payload: json.RawMessage(`{"metrics":{}}`)}},
		applyResult:   ApplyActionResult{Seq: 2, Delta: json.RawMessage(`{"ops":[]}`), WinPayload: json.RawMessage(`{"won":false}`)},
	}

	h := NewWSHandler(HandlerConfig{Secret: secret, Runtime: runtime})
	s := httptest.NewServer(h)
	defer s.Close()

	conn := mustDial(t, s.URL, runID)
	defer conn.Close()
	mustWrite(t, conn, ws.Envelope{ProtocolVersion: ws.ProtocolVersion1, Type: ws.TypeHello, RunID: runID.String(), Payload: mustRawMsg(t, ws.HelloPayload{RunToken: tok})})
	_ = mustRead(t, conn)
	_ = mustRead(t, conn)

	mustWrite(t, conn, ws.Envelope{ProtocolVersion: ws.ProtocolVersion1, Type: ws.TypeApplyAction, RunID: runID.String(), Payload: mustRawMsg(t, ws.ApplyActionPayload{ActionKey: "enable_singleflight", ClientRequestID: "6ba7b810-9dad-41d1-80b4-00c04fd430c8", ExpectedSeq: 1})})

	accepted := mustRead(t, conn)
	if accepted.Type != ws.TypeActionAccepted {
		t.Fatalf("accepted type=%q", accepted.Type)
	}
	delta := mustRead(t, conn)
	if delta.Type != ws.TypeDelta || delta.Seq == nil || *delta.Seq != 2 {
		t.Fatalf("delta=%+v", delta)
	}
	if runtime.applyCalls != 1 {
		t.Fatalf("applyCalls=%d", runtime.applyCalls)
	}
}

func TestWSHandlerClosesOnNonMonotonicConnectDeltas(t *testing.T) {
	runID := uuid.New()
	secret := "secret"
	tok := mustMintToken(t, secret, runID)

	runtime := &fakeRuntime{connectResult: ConnectResult{
		SnapshotRequired: false,
		Deltas: []ws.Delta{{Seq: 2, Payload: json.RawMessage(`{"ops":[]}`)}},
	}}

	h := NewWSHandler(HandlerConfig{Secret: secret, Runtime: runtime})
	s := httptest.NewServer(h)
	defer s.Close()

	conn := mustDial(t, s.URL, runID)
	defer conn.Close()
	mustWrite(t, conn, ws.Envelope{ProtocolVersion: ws.ProtocolVersion1, Type: ws.TypeHello, RunID: runID.String(), Payload: mustRawMsg(t, ws.HelloPayload{RunToken: tok})})
	_ = mustRead(t, conn)

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := conn.ReadMessage()
	if err == nil {
		t.Fatalf("expected close")
	}
}

func TestWSHandlerReplacesExistingConnectionForRun(t *testing.T) {
	runID := uuid.New()
	secret := "secret"
	tok := mustMintToken(t, secret, runID)

	runtime := &fakeRuntime{connectResult: ConnectResult{SnapshotRequired: true, Snapshot: ws.Delta{Seq: 1, Payload: json.RawMessage(`{"metrics":{}}`)}}}
	h := NewWSHandler(HandlerConfig{Secret: secret, Runtime: runtime})
	s := httptest.NewServer(h)
	defer s.Close()

	conn1 := mustDial(t, s.URL, runID)
	defer conn1.Close()
	mustWrite(t, conn1, ws.Envelope{ProtocolVersion: ws.ProtocolVersion1, Type: ws.TypeHello, RunID: runID.String(), Payload: mustRawMsg(t, ws.HelloPayload{RunToken: tok})})
	_ = mustRead(t, conn1)
	_ = mustRead(t, conn1)

	conn2 := mustDial(t, s.URL, runID)
	defer conn2.Close()
	mustWrite(t, conn2, ws.Envelope{ProtocolVersion: ws.ProtocolVersion1, Type: ws.TypeHello, RunID: runID.String(), Payload: mustRawMsg(t, ws.HelloPayload{RunToken: tok})})
	_ = mustRead(t, conn2)
	_ = mustRead(t, conn2)

	_ = conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := conn1.ReadMessage()
	if err == nil {
		t.Fatalf("expected first connection closed")
	}
}

func TestWSHandlerPongKeepsConnectionAlive(t *testing.T) {
	runID := uuid.New()
	secret := "secret"
	tok := mustMintToken(t, secret, runID)

	runtime := &fakeRuntime{connectResult: ConnectResult{SnapshotRequired: true, Snapshot: ws.Delta{Seq: 1, Payload: json.RawMessage(`{"metrics":{}}`)}}}
	h := NewWSHandler(HandlerConfig{Secret: secret, Runtime: runtime, PingInterval: 40 * time.Millisecond, PongTimeout: 120 * time.Millisecond})
	s := httptest.NewServer(h)
	defer s.Close()

	conn := mustDial(t, s.URL, runID)
	defer conn.Close()
	mustWrite(t, conn, ws.Envelope{ProtocolVersion: ws.ProtocolVersion1, Type: ws.TypeHello, RunID: runID.String(), Payload: mustRawMsg(t, ws.HelloPayload{RunToken: tok})})
	_ = mustRead(t, conn)
	_ = mustRead(t, conn)

	deadline := time.Now().Add(220 * time.Millisecond)
	for time.Now().Before(deadline) {
		msg := mustRead(t, conn)
		if msg.Type == ws.TypePing {
			mustWrite(t, conn, ws.Envelope{Type: ws.TypePong})
		}
	}
}

func mustMintToken(t *testing.T, secret string, runID uuid.UUID) string {
	t.Helper()
	tok, err := token.MintRunToken(secret, token.MintRunTokenInput{UserID: uuid.New(), RunID: runID, Engine: token.EngineA, TTL: time.Hour, Now: time.Now().UTC()})
	if err != nil {
		t.Fatalf("MintRunToken() err=%v", err)
	}
	return tok
}

func mustDial(t *testing.T, baseURL string, runID uuid.UUID) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(baseURL, "http") + "/ws/engineA/" + runID.String()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial() err=%v", err)
	}
	return conn
}

func mustWrite(t *testing.T, c *websocket.Conn, v any) {
	t.Helper()
	if err := c.WriteJSON(v); err != nil {
		t.Fatalf("WriteJSON() err=%v", err)
	}
}

func mustRead(t *testing.T, c *websocket.Conn) ws.Envelope {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, body, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() err=%v", err)
	}
	var env ws.Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("Unmarshal err=%v body=%s", err, string(body))
	}
	return env
}

func mustRawMsg(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal() err=%v", err)
	}
	return b
}

func mustDecode(t *testing.T, in json.RawMessage, out any) {
	t.Helper()
	if err := json.Unmarshal(in, out); err != nil {
		t.Fatalf("Unmarshal payload err=%v", err)
	}
}

type fakeRuntime struct {
	mu            sync.Mutex
	connectResult ConnectResult
	applyResult   ApplyActionResult
	connectCalls  int
	applyCalls    int
}

func (f *fakeRuntime) Connect(_ context.Context, _ uuid.UUID, _ *int) (ConnectResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connectCalls++
	return f.connectResult, nil
}

func (f *fakeRuntime) ApplyAction(_ context.Context, _ uuid.UUID, _ ApplyActionInput) (ApplyActionResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.applyCalls++
	return f.applyResult, nil
}
