package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	enginerepo "github.com/sidsharma96/SysEscape/internal/engine/a/repo"
	engineasvc "github.com/sidsharma96/SysEscape/internal/engine/a/service"
	"github.com/sidsharma96/SysEscape/internal/engine/a/transport"
	"github.com/sidsharma96/SysEscape/internal/platform/storage"
	"github.com/sidsharma96/SysEscape/internal/ws"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

type EngineARuntimeConfig struct {
	DB          *pgxpool.Pool
	RunRepo     *enginerepo.PostgresRunRepo
	BundleStore storage.BundleStore
}

type EngineARuntime struct {
	db      *pgxpool.Pool
	runRepo *enginerepo.PostgresRunRepo
	store   storage.BundleStore

	mu     sync.Mutex
	states map[uuid.UUID]*runState
}

type runState struct {
	mu sync.Mutex

	runID   uuid.UUID
	engine  *engineasvc.Engine
	lastSeq int

	tickInterval  time.Duration
	durationTicks int
	nextTickAt    time.Time
	terminal      bool
	sentWin       bool
	tickStarted   bool

	deltaSubs map[chan ws.Delta]struct{}
	winSubs   map[chan json.RawMessage]struct{}
}

func NewEngineARuntime(cfg EngineARuntimeConfig) *EngineARuntime {
	return &EngineARuntime{
		db:      cfg.DB,
		runRepo: cfg.RunRepo,
		store:   cfg.BundleStore,
		states:  make(map[uuid.UUID]*runState),
	}
}

func (r *EngineARuntime) Connect(ctx context.Context, runID uuid.UUID, _ *int) (transport.ConnectResult, error) {
	s, err := r.state(ctx, runID)
	if err != nil {
		return transport.ConnectResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r.ensureTickLoopLocked(s)
	payload, _ := json.Marshal(s.engine.Snapshot())
	return transport.ConnectResult{
		SnapshotRequired: true,
		Snapshot:         ws.Delta{Seq: s.lastSeq, Payload: payload},
	}, nil
}

func (r *EngineARuntime) ApplyAction(ctx context.Context, runID uuid.UUID, in transport.ApplyActionInput) (transport.ApplyActionResult, error) {
	s, err := r.state(ctx, runID)
	if err != nil {
		return transport.ApplyActionResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.terminal {
		return transport.ApplyActionResult{}, fmt.Errorf("run completed")
	}
	req := in.ClientRequestID
	action, err := r.runRepo.AppendAction(ctx, enginerepo.AppendActionInput{
		RunID:           runID,
		ActionType:      models.RunActionTypePlayer,
		ActionKey:       &in.ActionKey,
		ClientRequestID: &req,
	})
	if err != nil {
		return transport.ApplyActionResult{}, err
	}
	if action.Seq > s.lastSeq {
		if err := s.engine.ApplyAction(in.ActionKey, &req); err != nil {
			return transport.ApplyActionResult{}, err
		}
		s.lastSeq = action.Seq
	}
	payload, _ := json.Marshal(s.engine.Snapshot())
	s.nextTickAt = time.Now().Add(s.tickInterval)
	r.ensureTickLoopLocked(s)
	return transport.ApplyActionResult{Seq: action.Seq, Delta: payload}, nil
}

func (r *EngineARuntime) SubscribeDeltas(ctx context.Context, runID uuid.UUID) (<-chan ws.Delta, error) {
	s, err := r.state(ctx, runID)
	if err != nil {
		return nil, err
	}
	ch := make(chan ws.Delta, 16)
	s.mu.Lock()
	s.deltaSubs[ch] = struct{}{}
	r.ensureTickLoopLocked(s)
	s.mu.Unlock()
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		delete(s.deltaSubs, ch)
		close(ch)
		s.mu.Unlock()
	}()
	return ch, nil
}

func (r *EngineARuntime) SubscribeWinUpdates(ctx context.Context, runID uuid.UUID) (<-chan json.RawMessage, error) {
	s, err := r.state(ctx, runID)
	if err != nil {
		return nil, err
	}
	ch := make(chan json.RawMessage, 4)
	s.mu.Lock()
	s.winSubs[ch] = struct{}{}
	s.mu.Unlock()
	return ch, nil
}

func (r *EngineARuntime) tickLoop(s *runState) {
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		if s.terminal {
			s.tickStarted = false
			s.mu.Unlock()
			return
		}
		if !s.nextTickAt.IsZero() && time.Now().Before(s.nextTickAt) {
			s.mu.Unlock()
			continue
		}
		action, err := r.runRepo.AppendAction(context.Background(), enginerepo.AppendActionInput{
			RunID:      s.runID,
			ActionType: models.RunActionTypeTick,
		})
		if err != nil || s.engine.Tick() != nil {
			s.tickStarted = false
			s.mu.Unlock()
			return
		}
		s.lastSeq = action.Seq
		snap := s.engine.Snapshot()
		timeout := s.durationTicks > 0 && snap.Tick >= s.durationTicks
		if snap.Won || timeout {
			s.terminal = true
			s.tickStarted = false
			if !s.sentWin {
				s.sentWin = true
				win, _ := json.Marshal(map[string]any{"won": snap.Won, "timeout": timeout, "tick": snap.Tick})
				for ch := range s.winSubs {
					select {
					case ch <- win:
					default:
					}
				}
			}
			s.mu.Unlock()
			return
		}
		payload, _ := json.Marshal(snap)
		pushDelta(s.deltaSubs, ws.Delta{Seq: action.Seq, Payload: payload})
		s.mu.Unlock()
	}
}

func (r *EngineARuntime) state(ctx context.Context, runID uuid.UUID) (*runState, error) {
	r.mu.Lock()
	if s := r.states[runID]; s != nil {
		r.mu.Unlock()
		return s, nil
	}
	r.mu.Unlock()

	run, err := r.runRepo.GetRunByID(ctx, runID)
	if err != nil || run == nil {
		return nil, fmt.Errorf("run not found")
	}

	var bundleHash string
	if err := r.db.QueryRow(ctx, `SELECT COALESCE(bundle_hash, '') FROM room_versions WHERE id = $1`, run.RoomVersionID).Scan(&bundleHash); err != nil {
		return nil, err
	}
	if bundleHash == "" {
		return nil, fmt.Errorf("missing bundle hash")
	}

	rc, err := r.store.Download(ctx, bundleHash)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return nil, err
	}
	bundle, err := engineasvc.LoadEngineABundleFromTar(raw)
	if err != nil {
		return nil, err
	}

	actions, err := r.runRepo.ListActions(ctx, runID)
	if err != nil {
		return nil, err
	}
	if len(actions) == 0 {
		a, err := r.runRepo.AppendAction(ctx, enginerepo.AppendActionInput{RunID: runID, ActionType: models.RunActionTypeTick})
		if err != nil {
			return nil, err
		}
		actions = append(actions, *a)
	}

	logEntries := make([]engineasvc.LogEntry, 0, len(actions))
	lastSeq := 0
	for _, a := range actions {
		logEntries = append(logEntries, engineasvc.LogEntry{
			Seq:             a.Seq,
			ActionType:      string(a.ActionType),
			ActionKey:       a.ActionKey,
			ClientRequestID: a.ClientRequestID,
		})
		if a.Seq > lastSeq {
			lastSeq = a.Seq
		}
	}
	engine, err := engineasvc.Replay(bundle.Simulation, logEntries)
	if err != nil {
		return nil, err
	}

	s := &runState{
		runID:         runID,
		engine:        engine,
		lastSeq:       lastSeq,
		tickInterval:  max(time.Duration(bundle.Simulation.TickIntervalMS)*time.Millisecond, 20*time.Millisecond),
		durationTicks: bundle.Simulation.DurationTicks,
		terminal:      engine.Snapshot().Won || (bundle.Simulation.DurationTicks > 0 && engine.Snapshot().Tick >= bundle.Simulation.DurationTicks),
		deltaSubs:     make(map[chan ws.Delta]struct{}),
		winSubs:       make(map[chan json.RawMessage]struct{}),
	}
	r.mu.Lock()
	r.states[runID] = s
	r.mu.Unlock()
	return s, nil
}

func (r *EngineARuntime) ensureTickLoopLocked(s *runState) {
	if s.tickStarted || s.terminal {
		return
	}
	s.tickStarted = true
	go r.tickLoop(s)
}

func pushDelta(subs map[chan ws.Delta]struct{}, d ws.Delta) {
	for ch := range subs {
		select {
		case ch <- d:
		default:
		}
	}
}
