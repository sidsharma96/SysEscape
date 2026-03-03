package transport

import (
	"context"
	"encoding/json"
	"errors"
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

const (
	engineAWSPrefix = "/ws/engineA/"

	closeExpectedHello      = 4001
	closeInvalidHello       = 4002
	closeRunIDMismatch      = 4003
	closeUnauthorized       = 4004
	closeSeqGap             = 4005
	closeBadMessage         = 4006
	closeBadApplyAction     = 4007
	closeConnectionReplaced = 4009
	closeInternal           = 4010
)

var errConnClosed = errors.New("websocket connection closed")

type Runtime interface {
	Connect(ctx context.Context, runID uuid.UUID, resumeFromSeq *int) (ConnectResult, error)
	ApplyAction(ctx context.Context, runID uuid.UUID, in ApplyActionInput) (ApplyActionResult, error)
	SubscribeDeltas(ctx context.Context, runID uuid.UUID) (<-chan ws.Delta, error)
}

type winUpdateRuntime interface {
	SubscribeWinUpdates(ctx context.Context, runID uuid.UUID) (<-chan json.RawMessage, error)
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
	active map[uuid.UUID]*connWriter
}

type connWriter struct {
	mu     sync.Mutex
	conn   *websocket.Conn
	closed bool
}

type seqTracker struct {
	mu   sync.Mutex
	last int
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
		active: make(map[uuid.UUID]*connWriter),
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
	writer := &connWriter{conn: conn}
	defer writer.closeWith(websocket.CloseNormalClosure, "closing")

	env, err := readEnvelope(conn)
	if err != nil {
		writer.closeWith(closeBadMessage, "invalid first message")
		return
	}
	if env.Type != ws.TypeHello {
		writer.closeWith(closeExpectedHello, "expected hello")
		return
	}
	hello, err := ws.ValidateAndDecodeHello(env)
	if err != nil {
		writer.closeWith(closeInvalidHello, "invalid hello")
		return
	}
	if env.RunID != runID.String() {
		writer.closeWith(closeRunIDMismatch, "runId mismatch")
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
		writer.closeWith(closeUnauthorized, "unauthorized")
		return
	}

	h.registerActive(runID, writer)
	defer h.unregisterActive(runID, writer)

	connectResult, err := h.cfg.Runtime.Connect(ctx, runID, hello.ResumeFromSeq)
	if err != nil {
		writer.closeWith(closeInternal, "connect failed")
		return
	}
	if err := writer.write(ws.Envelope{
		Type:    ws.TypeHelloAck,
		Payload: mustRaw(ws.HelloAckPayload{SnapshotRequired: connectResult.SnapshotRequired}),
	}); err != nil {
		return
	}

	tracker := &seqTracker{}
	if connectResult.SnapshotRequired {
		if connectResult.Snapshot.Seq <= 0 {
			writer.closeWith(closeSeqGap, "invalid snapshot sequence")
			return
		}
		if err := writer.write(withDelta(ws.TypeSnapshot, connectResult.Snapshot)); err != nil {
			return
		}
		tracker.set(connectResult.Snapshot.Seq)
	} else {
		base := 0
		if hello.ResumeFromSeq != nil {
			base = *hello.ResumeFromSeq
		}
		tracker.set(base)
		for _, d := range connectResult.Deltas {
			ok, gap := tracker.advance(d.Seq)
			if gap {
				writer.closeWith(closeSeqGap, "resume sequence gap")
				return
			}
			if !ok {
				continue
			}
			if err := writer.write(withDelta(ws.TypeDelta, d)); err != nil {
				return
			}
		}
	}

	deltaCh, err := h.cfg.Runtime.SubscribeDeltas(ctx, runID)
	if err != nil {
		writer.closeWith(closeInternal, "subscribe failed")
		return
	}
	stopDelta := make(chan struct{})
	go h.streamRuntimeDeltas(stopDelta, writer, tracker, deltaCh)
	defer close(stopDelta)

	if runtimeWithWinUpdates, ok := h.cfg.Runtime.(winUpdateRuntime); ok {
		winCh, err := runtimeWithWinUpdates.SubscribeWinUpdates(ctx, runID)
		if err != nil {
			writer.closeWith(closeInternal, "subscribe win updates failed")
			return
		}
		stopWin := make(chan struct{})
		go h.streamRuntimeWinUpdates(stopWin, writer, winCh)
		defer close(stopWin)
	}

	hb := ws.NewHeartbeat(h.cfg.NowFn, h.cfg.PingInterval, h.cfg.PongTimeout)
	stopHeartbeat := make(chan struct{})
	go h.runHeartbeat(stopHeartbeat, hb, writer)
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
			in, err := ws.ValidateAndDecodeApplyAction(env)
			if err != nil {
				writer.closeWith(closeBadApplyAction, "invalid action")
				return
			}
			if env.RunID != runID.String() {
				writer.closeWith(closeRunIDMismatch, "runId mismatch")
				return
			}

			out, err := h.cfg.Runtime.ApplyAction(ctx, runID, ApplyActionInput{ActionKey: in.ActionKey, ClientRequestID: in.ClientRequestID, ExpectedSeq: in.ExpectedSeq})
			if err != nil {
				writer.closeWith(closeInternal, "apply action failed")
				return
			}
			if err := writer.write(ws.Envelope{Type: ws.TypeActionAccepted, Payload: mustRaw(ws.ActionAcceptedPayload{ActionKey: in.ActionKey, Seq: out.Seq})}); err != nil {
				return
			}
			sent, ok := h.writeDelta(writer, tracker, ws.Delta{Seq: out.Seq, Payload: out.Delta})
			if !ok {
				return
			}
			if sent && len(out.WinPayload) > 0 {
				if err := writer.write(ws.Envelope{Type: ws.TypeWinUpdate, Payload: out.WinPayload}); err != nil {
					return
				}
			}
		default:
			writer.closeWith(closeBadMessage, "unsupported message")
			return
		}
	}
}

func (h *wsHandler) streamRuntimeWinUpdates(stop <-chan struct{}, writer *connWriter, winCh <-chan json.RawMessage) {
	for {
		select {
		case <-stop:
			return
		case payload, ok := <-winCh:
			if !ok {
				return
			}
			if len(payload) == 0 {
				continue
			}
			if err := writer.write(ws.Envelope{Type: ws.TypeWinUpdate, Payload: payload}); err != nil {
				return
			}
		}
	}
}

func (h *wsHandler) streamRuntimeDeltas(stop <-chan struct{}, writer *connWriter, tracker *seqTracker, deltaCh <-chan ws.Delta) {
	for {
		select {
		case <-stop:
			return
		case d, ok := <-deltaCh:
			if !ok {
				return
			}
			_, ok = h.writeDelta(writer, tracker, d)
			if !ok {
				return
			}
		}
	}
}

func (h *wsHandler) writeDelta(writer *connWriter, tracker *seqTracker, delta ws.Delta) (sent bool, ok bool) {
	if delta.Seq <= 0 {
		writer.closeWith(closeSeqGap, "invalid sequence")
		return false, false
	}
	send, gap := tracker.advance(delta.Seq)
	if gap {
		writer.closeWith(closeSeqGap, "sequence gap")
		return false, false
	}
	if !send {
		return false, true
	}
	if err := writer.write(withDelta(ws.TypeDelta, delta)); err != nil {
		return false, false
	}
	return true, true
}

func (h *wsHandler) runHeartbeat(stop <-chan struct{}, hb *ws.Heartbeat, writer *connWriter) {
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
				writer.closeWith(closeBadMessage, "pong timeout")
				return
			}
			if hb.PingDue(lastPingAt) {
				if err := writer.write(ws.Envelope{Type: ws.TypePing}); err != nil {
					return
				}
				lastPingAt = h.cfg.NowFn()
			}
		}
	}
}

func (h *wsHandler) registerActive(runID uuid.UUID, writer *connWriter) {
	h.mu.Lock()
	old := h.active[runID]
	h.active[runID] = writer
	h.mu.Unlock()
	if old != nil && old != writer {
		old.closeWith(closeConnectionReplaced, "connection replaced")
	}
}

func (h *wsHandler) unregisterActive(runID uuid.UUID, writer *connWriter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.active[runID] == writer {
		delete(h.active, runID)
	}
}

func (w *connWriter) write(env ws.Envelope) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return errConnClosed
	}
	return w.conn.WriteJSON(env)
}

func (w *connWriter) closeWith(code int, reason string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	_ = w.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
	_ = w.conn.Close()
	w.closed = true
}

func (s *seqTracker) set(seq int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = seq
}

func (s *seqTracker) advance(seq int) (send bool, gap bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if seq <= s.last {
		return false, false
	}
	if seq != s.last+1 {
		return false, true
	}
	s.last = seq
	return true, false
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
