package ws

import (
	"testing"
	"time"
)

func TestHeartbeatDefaults(t *testing.T) {
	if DefaultPingInterval != 25*time.Second {
		t.Fatalf("DefaultPingInterval=%s", DefaultPingInterval)
	}
	if DefaultPongTimeout != 75*time.Second {
		t.Fatalf("DefaultPongTimeout=%s", DefaultPongTimeout)
	}
}

func TestHeartbeatPingAndTimeout(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	hb := NewHeartbeat(clock.Now, DefaultPingInterval, DefaultPongTimeout)

	lastPing := clock.Now()
	if hb.PingDue(lastPing) {
		t.Fatalf("PingDue() true too early")
	}

	clock.Advance(DefaultPingInterval)
	if !hb.PingDue(lastPing) {
		t.Fatalf("PingDue() false at interval")
	}

	clock.Advance(DefaultPongTimeout + time.Second)
	if !hb.TimedOut() {
		t.Fatalf("TimedOut() false after timeout")
	}
}

func TestHeartbeatRecordPongResetsTimeout(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	hb := NewHeartbeat(clock.Now, time.Second, 3*time.Second)

	clock.Advance(2 * time.Second)
	hb.RecordPong()
	clock.Advance(2 * time.Second)
	if hb.TimedOut() {
		t.Fatalf("TimedOut() true after pong reset")
	}
}

type fakeClock struct {
	now time.Time
}

func (f *fakeClock) Now() time.Time { return f.now }

func (f *fakeClock) Advance(d time.Duration) { f.now = f.now.Add(d) }
