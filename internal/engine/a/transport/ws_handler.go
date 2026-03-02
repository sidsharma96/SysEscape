package transport

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	platformlog "github.com/sidsharma96/SysEscape/internal/platform/log"
	"github.com/sidsharma96/SysEscape/internal/token"
	"github.com/sidsharma96/SysEscape/internal/ws"
)

const engineAWSPrefix = "/ws/engineA/"

type Runtime interface {
	Connect(ctx context.Context, runID uuid.UUID, resumeFromSeq *int) (ConnectResult, error)
	ApplyAction(ctx context.Context, runID uuid.UUID, in ApplyActionInput) (ApplyActionResult, error)
}

type ConnectResult struct {
	SnapshotRequired bool
	Snapshot         ws.Delta
	Deltas           []ws.Delta
}

type ApplyActionInput struct {
	ActionKey       string
	ClientRequestID string
	ExpectedSeq     int
}

type ApplyActionResult struct {
	Seq        int
	Delta      json.RawMessage
	WinPayload json.RawMessage
}

type HandlerConfig struct {
	Secret       string
	Runtime      Runtime
	PingInterval time.Duration
	PongTimeout  time.Duration
	NowFn        func() time.Time
}

type wsHandler struct {
	cfg      HandlerConfig
	logger   *slog.Logger
	upgrader websocket.Upgrader

	mu     sync.Mutex
	active map[uuid.UUID]*websocket.Conn
}

type connWriter struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func NewWSHandler(cfg HandlerConfig) http.Handler {
	if cfg.PingInterval <= 0 {
		cfg.PingInterval = ws.DefaultPingInterval
	}
	if cfg.PongTimeout <= 0 {
		cfg.PongTimeout = ws.DefaultPongTimeout
	}
	if cfg.NowFn == nil {
		cfg.NowFn = time.Now
	}
	return &wsHandler{
		cfg:    cfg,
		logger: platformlog.NewLogger(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
		active: make(map[uuid.UUID]*websocket.Conn),
	}
}

func (h *wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseRunID(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if h.cfg.Runtime == nil || strings.TrimSpace(h.cfg.Secret) == "" {
		http.Error(w, "ws handler not configured", http.StatusInternalServerError)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.handleConnection(r.Context(), runID, conn)
}

func (h *wsHandler) handleConnection(ctx context.Context, runID uuid.UUID, conn *websocket.Conn) {
	defer conn.Close()

	writer := &connWriter{conn: conn}
	env, err := readEnvelope(conn)
	if err != nil {
		return
	}
	if env.Type != ws.TypeHello {
		return
	}
	if err := ws.ValidateClientEnvelope(env); err != nil {
		return
	}
	if env.RunID != runID.String() {
		return
	}

	hello, err := ws.DecodeHelloPayload(env.Payload)
	if err != nil {
		return
	}
	if _, err := token.VerifyRunToken(token.VerifyRunTokenInput{
		Token:          hello.RunToken,
		Secret:         h.cfg.Secret,
		ExpectedRunID:  runID,
		ExpectedEngine: token.EngineA,
		Now:            h.cfg.NowFn().UTC(),
	}); err != nil {
		h.logger.Warn("ws hello token verify failed", slog.String("runId", runID.String()), slog.String("error", err.Error()))
		return
	}

	h.registerActive(runID, conn)
	defer h.unregisterActive(runID, conn)

	connectResult, err := h.cfg.Runtime.Connect(ctx, runID, hello.ResumeFromSeq)
	if err != nil {
		return
	}
	if err := writer.write(ws.Envelope{
		Type:    ws.TypeHelloAck,
		Payload: mustRaw(ws.HelloAckPayload{SnapshotRequired: connectResult.SnapshotRequired}),
	}); err != nil {
		return
	}

	if connectResult.SnapshotRequired {
		if err := writer.write(withDelta(ws.TypeSnapshot, connectResult.Snapshot)); err != nil {
			return
		}
	} else {
		expected := 1
		if hello.ResumeFromSeq != nil {
			expected = *hello.ResumeFromSeq + 1
		}
		for _, d := range connectResult.Deltas {
			if d.Seq != expected {
				return
			}
			if err := writer.write(withDelta(ws.TypeDelta, d)); err != nil {
				return
			}
			expected++
		}
	}

	hb := ws.NewHeartbeat(h.cfg.NowFn, h.cfg.PingInterval, h.cfg.PongTimeout)
	stopHeartbeat := make(chan struct{})
	go h.runHeartbeat(stopHeartbeat, hb, writer, conn)
	defer close(stopHeartbeat)

	for {
		env, err := readEnvelope(conn)
		if err != nil {
			return
		}
		switch env.Type {
		case ws.TypePong:
			hb.RecordPong()
		case ws.TypeApplyAction:
			if err := ws.ValidateClientEnvelope(env); err != nil {
				return
			}
			if env.RunID != runID.String() {
				return
			}
			in, err := ws.DecodeApplyActionPayload(env.Payload)
			if err != nil {
				return
			}

			out, err := h.cfg.Runtime.ApplyAction(ctx, runID, ApplyActionInput{ActionKey: in.ActionKey, ClientRequestID: in.ClientRequestID, ExpectedSeq: in.ExpectedSeq})
			if err != nil {
				return
			}
			if err := writer.write(ws.Envelope{Type: ws.TypeActionAccepted, Payload: mustRaw(ws.ActionAcceptedPayload{ActionKey: in.ActionKey, Seq: out.Seq})}); err != nil {
				return
			}
			d := ws.Delta{Seq: out.Seq, Payload: out.Delta}
			if err := writer.write(withDelta(ws.TypeDelta, d)); err != nil {
				return
			}
			if len(out.WinPayload) > 0 {
				if err := writer.write(ws.Envelope{Type: ws.TypeWinUpdate, Payload: out.WinPayload}); err != nil {
					return
				}
			}
		default:
			return
		}
	}
}

func (h *wsHandler) runHeartbeat(stop <-chan struct{}, hb *ws.Heartbeat, writer *connWriter, conn *websocket.Conn) {
	tickEvery := h.cfg.PingInterval / 4
	if tickEvery < 10*time.Millisecond {
		tickEvery = 10 * time.Millisecond
	}
	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()

	lastPingAt := h.cfg.NowFn()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if hb.TimedOut() {
				_ = conn.Close()
				return
			}
			if hb.PingDue(lastPingAt) {
				if err := writer.write(ws.Envelope{Type: ws.TypePing}); err != nil {
					_ = conn.Close()
					return
				}
				lastPingAt = h.cfg.NowFn()
			}
		}
	}
}

func (h *wsHandler) registerActive(runID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	old := h.active[runID]
	h.active[runID] = conn
	h.mu.Unlock()
	if old != nil && old != conn {
		_ = old.Close()
	}
}

func (h *wsHandler) unregisterActive(runID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.active[runID] == conn {
		delete(h.active, runID)
	}
}

func (w *connWriter) write(env ws.Envelope) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(env)
}

func readEnvelope(conn *websocket.Conn) (ws.Envelope, error) {
	_, data, err := conn.ReadMessage()
	if err != nil {
		return ws.Envelope{}, err
	}
	return ws.DecodeEnvelope(data)
}

func parseRunID(path string) (uuid.UUID, bool) {
	if !strings.HasPrefix(path, engineAWSPrefix) {
		return uuid.Nil, false
	}
	raw := strings.TrimPrefix(path, engineAWSPrefix)
	if raw == "" || strings.Contains(raw, "/") {
		return uuid.Nil, false
	}
	runID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return runID, true
}

func withDelta(msgType string, delta ws.Delta) ws.Envelope {
	seq := delta.Seq
	return ws.Envelope{Type: msgType, Seq: &seq, Payload: delta.Payload}
}

func mustRaw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
