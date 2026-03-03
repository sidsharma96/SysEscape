// smoke_m3.go — E2E smoke test for M3 Engine A WebSocket flow.
//
// Requires a running local stack (make dev-up + services).
// Reads env: SER_BFF_URL, ENGINE_A_WS_URL
//
// Sequence:
//  1. Health check on BFF
//  2. WS hello → hello_ack
//  3. Wait for snapshot (seq 1)
//  4. Wait for 2 deltas
//  5. Send apply_action → assert action_accepted
//  6. Clean close
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type envelope struct {
	ProtocolVersion int    `json:"protocolVersion,omitempty"`
	Type            string `json:"type"`
	RunID           string `json:"runId,omitempty"`
	Seq             *int   `json:"seq,omitempty"`
	Payload         any    `json:"payload,omitempty"`
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fatal(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "smoke-m3 FAIL: "+msg+"\n", args...)
	os.Exit(1)
}

func readMsg(c *websocket.Conn, timeout time.Duration) envelope {
	c.SetReadDeadline(time.Now().Add(timeout))
	_, raw, err := c.ReadMessage()
	if err != nil {
		fatal("read: %v", err)
	}
	var msg envelope
	if err := json.Unmarshal(raw, &msg); err != nil {
		fatal("unmarshal: %v", err)
	}
	// Auto-reply to pings
	if msg.Type == "ping" {
		pong := envelope{Type: "pong"}
		b, _ := json.Marshal(pong)
		c.WriteMessage(websocket.TextMessage, b)
		return readMsg(c, timeout)
	}
	return msg
}

func main() {
	bffURL := env("SER_BFF_URL", "http://localhost:8080")
	wsURL := env("ENGINE_A_WS_URL", "ws://localhost:8081")

	// 1. Healthz
	resp, err := http.Get(bffURL + "/healthz")
	if err != nil {
		fatal("healthz request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fatal("healthz status: %d", resp.StatusCode)
	}
	fmt.Println("  healthz OK")

	// 2. Connect WS with a synthetic run
	runID := uuid.NewString()
	runToken := uuid.NewString()
	u, _ := url.Parse(wsURL + "/ws/engineA/" + runID)

	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		fatal("ws dial: %v", err)
	}
	defer c.Close()

	// Send hello
	hello := envelope{
		ProtocolVersion: 1,
		Type:            "hello",
		RunID:           runID,
		Payload:         map[string]any{"runToken": runToken},
	}
	helloBytes, _ := json.Marshal(hello)
	if err := c.WriteMessage(websocket.TextMessage, helloBytes); err != nil {
		fatal("send hello: %v", err)
	}

	// 3. Expect hello_ack
	ack := readMsg(c, 5*time.Second)
	if ack.Type != "hello_ack" {
		fatal("expected hello_ack, got %s", ack.Type)
	}
	fmt.Println("  hello_ack OK")

	// 4. Wait for snapshot
	snap := readMsg(c, 10*time.Second)
	if snap.Type != "snapshot" {
		fatal("expected snapshot, got %s", snap.Type)
	}
	if snap.Seq == nil {
		fatal("snapshot missing seq")
	}
	lastSeq := *snap.Seq
	fmt.Printf("  snapshot OK (seq=%d)\n", lastSeq)

	// 5. Wait for 2 deltas
	for i := 0; i < 2; i++ {
		delta := readMsg(c, 15*time.Second)
		if delta.Type != "delta" {
			fatal("expected delta, got %s", delta.Type)
		}
		if delta.Seq == nil {
			fatal("delta missing seq")
		}
		lastSeq = *delta.Seq
		fmt.Printf("  delta OK (seq=%d)\n", lastSeq)
	}

	// 6. Send apply_action
	action := envelope{
		ProtocolVersion: 1,
		Type:            "apply_action",
		RunID:           runID,
		Payload: map[string]any{
			"actionKey":       "restart_nginx",
			"clientRequestId": uuid.NewString(),
			"expectedSeq":     lastSeq,
		},
	}
	actionBytes, _ := json.Marshal(action)
	if err := c.WriteMessage(websocket.TextMessage, actionBytes); err != nil {
		fatal("send apply_action: %v", err)
	}

	// Wait for action_accepted (may receive deltas first)
	deadline := time.Now().Add(10 * time.Second)
	accepted := false
	for time.Now().Before(deadline) {
		msg := readMsg(c, time.Until(deadline))
		if msg.Type == "action_accepted" {
			accepted = true
			fmt.Println("  action_accepted OK")
			break
		}
		// Skip deltas/snapshots while waiting
	}
	if !accepted {
		fatal("never received action_accepted")
	}

	// 7. Clean close
	err = c.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	if err != nil {
		fatal("close: %v", err)
	}

	fmt.Println("smoke-m3 passed")
}
