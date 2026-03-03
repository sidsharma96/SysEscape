package main

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	enginerepo "github.com/sidsharma96/SysEscape/internal/engine/a/repo"
	"github.com/sidsharma96/SysEscape/internal/engine/a/transport"
	"github.com/sidsharma96/SysEscape/internal/platform/storage"
	"github.com/sidsharma96/SysEscape/internal/token"
	"github.com/sidsharma96/SysEscape/internal/ws"
)

func TestIntegrationEngineAWSFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped in short mode")
	}

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv(): %v", err)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil || pool.Ping(context.Background()) != nil {
		t.Skipf("skip: postgres unavailable: %v", err)
	}
	defer pool.Close()

	store, err := storage.NewS3BundleStore(storage.StorageConfig{
		Endpoint: cfg.S3Endpoint, Bucket: cfg.S3Bucket, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey, Region: cfg.S3Region, ForcePathStyle: cfg.S3ForcePathStyle,
	})
	if err != nil {
		t.Skipf("skip: s3 unavailable: %v", err)
	}
	hash := strings.Repeat("a", 64)
	bundle := buildBundleTar(t)
	if err := store.Upload(context.Background(), hash, bytes.NewReader(bundle), int64(len(bundle))); err != nil {
		t.Skipf("skip: cannot upload bundle: %v", err)
	}

	userID, runID, roomID, rvID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	slug := "engine-a-it-" + uuid.NewString()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, github_id, github_username, display_name, role) VALUES ($1,$2,$3,$4,'USER')`, userID, time.Now().UnixNano(), "enginea_it_"+slug, "Engine A IT"); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO rooms (id, slug, title, district, engine, difficulty, description) VALUES ($1,$2,'it room','distributed-systems','A','L1','it')`, roomID, slug); err != nil {
		t.Fatalf("seed room: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO room_versions (id, room_id, version_number, status, bundle_hash, changelog) VALUES ($1,$2,1,'PUBLISHED',$3,'it')`, rvID, roomID, hash); err != nil {
		t.Fatalf("seed room_version: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO runs (id, user_id, room_version_id, seed, status) VALUES ($1,$2,$3,42,'ACTIVE')`, runID, userID, rvID); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		pool.Exec(bg, `DELETE FROM run_actions WHERE run_id = $1`, runID)
		pool.Exec(bg, `DELETE FROM runs WHERE id = $1`, runID)
		pool.Exec(bg, `DELETE FROM room_versions WHERE id = $1`, rvID)
		pool.Exec(bg, `DELETE FROM rooms WHERE id = $1`, roomID)
		pool.Exec(bg, `DELETE FROM users WHERE id = $1`, userID)
	})

	rt := NewEngineARuntime(EngineARuntimeConfig{DB: pool, RunRepo: enginerepo.NewPostgresRunRepo(pool), BundleStore: store})
	srv := httptest.NewServer(transport.NewWSHandler(transport.HandlerConfig{Secret: cfg.RunTokenSecret, Runtime: rt}))
	defer srv.Close()

	tok, err := token.MintRunToken(cfg.RunTokenSecret, token.MintRunTokenInput{UserID: userID, RunID: runID, Engine: token.EngineA, TTL: time.Minute, Now: time.Now().UTC()})
	if err != nil {
		t.Fatalf("MintRunToken(): %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/engineA/"+runID.String(), nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	readEnv := func(timeout time.Duration) ws.Envelope {
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		_, body, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		var msg ws.Envelope
		if err := json.Unmarshal(body, &msg); err != nil {
			t.Fatalf("decode envelope: %v", err)
		}
		return msg
	}
	readDelta := func() map[string]float64 {
		for {
			msg := readEnv(3 * time.Second)
			if msg.Type != ws.TypeDelta {
				continue
			}
			var payload struct {
				Metrics map[string]float64 `json:"metrics"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				t.Fatalf("decode delta payload: %v", err)
			}
			return payload.Metrics
		}
	}
	writeEnv := func(v any) {
		if err := conn.WriteJSON(v); err != nil {
			t.Fatalf("WriteJSON: %v", err)
		}
	}

	helloPayload, _ := json.Marshal(ws.HelloPayload{RunToken: tok})
	writeEnv(ws.Envelope{ProtocolVersion: ws.ProtocolVersion1, Type: ws.TypeHello, RunID: runID.String(), Payload: helloPayload})

	if msg := readEnv(3 * time.Second); msg.Type != ws.TypeHelloAck {
		t.Fatalf("hello msg=%q", msg.Type)
	}
	snap := readEnv(3 * time.Second)
	if snap.Type != ws.TypeSnapshot || snap.Seq == nil {
		t.Fatalf("snapshot=%+v", snap)
	}

	d1 := readDelta()
	d2 := readDelta()
	if d2["pressure"] <= d1["pressure"] {
		t.Fatalf("expected baseline increase, d1=%v d2=%v", d1["pressure"], d2["pressure"])
	}

	actionPayload, _ := json.Marshal(ws.ApplyActionPayload{ActionKey: "cool_down", ClientRequestID: uuid.NewString(), ExpectedSeq: *snap.Seq + 2})
	writeEnv(ws.Envelope{ProtocolVersion: ws.ProtocolVersion1, Type: ws.TypeApplyAction, RunID: runID.String(), Payload: actionPayload})
	if msg := readEnv(3 * time.Second); msg.Type != ws.TypeActionAccepted {
		t.Fatalf("action msg=%q", msg.Type)
	}
	immediate := readEnv(3 * time.Second)
	if immediate.Type != ws.TypeDelta {
		t.Fatalf("immediate msg=%q", immediate.Type)
	}
	post := readDelta()
	if post["pressure"] >= d2["pressure"] {
		t.Fatalf("expected reversed direction after action, post=%v before=%v", post["pressure"], d2["pressure"])
	}
	win := readEnv(3 * time.Second)
	if win.Type != ws.TypeWinUpdate {
		t.Fatalf("win msg=%q", win.Type)
	}
	trailing := 0
	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, body, err := conn.ReadMessage()
		if err != nil {
			var netErr interface{ Timeout() bool }
			if errors.As(err, &netErr) && netErr.Timeout() {
				break
			}
			t.Fatalf("unexpected read error: %v", err)
		}
		var env ws.Envelope
		_ = json.Unmarshal(body, &env)
		if env.Type == ws.TypeDelta {
			trailing++
		}
		if trailing > 1 {
			t.Fatalf("tick loop did not stop; trailing deltas=%d", trailing)
		}
	}
}

func buildBundleTar(t *testing.T) []byte {
	t.Helper()
	files := []struct {
		name string
		body string
	}{
		{"metadata.yaml", "slug: it-room\ntitle: IT Room\nengine: A\ndifficulty: L1\ndistrict: distributed-systems\n"},
		{"engineA/scenario.yaml", "topology: []\ninitialMetrics:\n  pressure: 10\nconfig: {}\n"},
		{"engineA/actions.yaml", "actions:\n  - key: cool_down\n    description: cool down\n"},
		{"engineA/signals.yaml", "metrics: [pressure]\nlogPatterns: []\ntraceSpans: []\n"},
		{"engineA/win_checks.yaml", "checks:\n  - metric: pressure\n    op: \"<\"\n    value: 5\n"},
		{"engineA/simulation.yaml", "simulation:\n  tick_interval_ms: 30\n  duration_ticks: 12\n  events:\n    - at_tick: 0\n      effects:\n        - metric: pressure\n          target: 100\n          rate: 10\n  action_effects:\n    cool_down:\n      effects:\n        - metric: pressure\n          target: 0\n          rate: 30\n"},
	}
	var out bytes.Buffer
	tw := tar.NewWriter(&out)
	for _, f := range files {
		b := []byte(f.body)
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o644, Typeflag: tar.TypeReg, Size: int64(len(b))}); err != nil {
			t.Fatalf("tar header: %v", err)
		}
		if _, err := tw.Write(b); err != nil {
			t.Fatalf("tar write: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	return out.Bytes()
}
