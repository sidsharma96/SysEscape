package service

import (
	"errors"
	"testing"
)

func TestWinTransitionStopsRun(t *testing.T) {
	e := NewEngine(loadSpec(t))
	must(t, e.ApplyAction("enable_singleflight", strptr("r1")))
	must(t, e.Tick())
	must(t, e.ApplyAction("enable_stale_while_revalidate", strptr("r2")))
	must(t, e.Tick())
	must(t, e.Tick())
	must(t, e.Tick())
	if !e.Snapshot().Won {
		t.Fatalf("expected win")
	}
	if !errors.Is(e.Tick(), ErrRunCompleted) || !errors.Is(e.ApplyAction("enable_singleflight", strptr("r3")), ErrRunCompleted) {
		t.Fatalf("expected run completed errors")
	}
}
