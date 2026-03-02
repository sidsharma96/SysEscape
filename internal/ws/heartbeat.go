package ws

import (
	"sync"
	"time"
)

const (
	DefaultPingInterval = 25 * time.Second
	DefaultPongTimeout  = 75 * time.Second
)

type Heartbeat struct {
	mu           sync.Mutex
	nowFn        func() time.Time
	pingInterval time.Duration
	pongTimeout  time.Duration
	lastPongAt   time.Time
}

func NewHeartbeat(nowFn func() time.Time, pingInterval, pongTimeout time.Duration) *Heartbeat {
	if nowFn == nil {
		nowFn = time.Now
	}
	if pingInterval <= 0 {
		pingInterval = DefaultPingInterval
	}
	if pongTimeout <= 0 {
		pongTimeout = DefaultPongTimeout
	}
	now := nowFn()
	return &Heartbeat{nowFn: nowFn, pingInterval: pingInterval, pongTimeout: pongTimeout, lastPongAt: now}
}

func (h *Heartbeat) RecordPong() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastPongAt = h.nowFn()
}

func (h *Heartbeat) PingDue(lastPingAt time.Time) bool {
	return h.nowFn().Sub(lastPingAt) >= h.pingInterval
}

func (h *Heartbeat) TimedOut() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.nowFn().Sub(h.lastPongAt) > h.pongTimeout
}
