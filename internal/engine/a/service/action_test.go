package service

import (
	"errors"
	"testing"
)

func TestActionOverrideUnknownAndTickShape(t *testing.T) {
	e := NewEngine(loadSpec(t))
	if err := e.ApplyAction("nope", strptr("x")); !errors.Is(err, ErrUnknownAction) {
		t.Fatalf("got %v", err)
	}
	must(t, e.ApplyAction("enable_singleflight", strptr("r1")))
	before := e.Snapshot().Metrics["db_connections"]
	must(t, e.Tick())
	after := e.Snapshot().Metrics["db_connections"]
	if after >= before {
		t.Fatalf("expected action override: before=%v after=%v", before, after)
	}
	l := e.Log()[1]
	if l.ActionType != ActionTypeTick || l.ActionKey != nil || l.ClientRequestID != nil || l.Seq != 2 {
		t.Fatalf("bad tick log: %+v", l)
	}
}
